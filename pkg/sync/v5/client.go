package sync

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/retry"

	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
)

// updateDynamicClient updates the client's server API group resource
// information and dynamic client pool.
func (s *sync) updateDynamicClient() error {
	grs, err := discovery.GetAPIGroupResources(s.cli)
	if err != nil {
		return err
	}
	s.grs = grs

	rm := discovery.NewRESTMapper(s.grs, meta.InterfacesForUnstructured)
	s.dyn = dynamic.NewClientPool(s.restconfig, rm, dynamic.LegacyAPIPathResolverFunc)

	return nil
}

// applyResources creates or updates all resources in db that match the provided
// filter.
func (s *sync) applyResources(filter func(unstructured.Unstructured) bool, keys []string) error {
	for _, k := range keys {
		o := s.db[k]

		if !filter(o) {
			continue
		}

		if err := s.write(&o); err != nil {
			return err
		}
	}
	return nil
}

// deleteOrphans looks for the "belongs-to-syncpod: yes" annotation, if found
// and object not in current db, remove it.
func (s *sync) deleteOrphans() error {
	s.log.Info("Deleting orphan objects from the running cluster")
	done := map[schema.GroupKind]struct{}{}

	for _, gr := range s.grs {
		for version, resources := range gr.VersionedResources {
			for _, resource := range resources {
				if strings.ContainsRune(resource.Name, '/') { // no subresources
					continue
				}

				if !contains(resource.Verbs, "list") {
					continue
				}

				gvk := schema.GroupVersionKind{Group: gr.Group.Name, Version: version, Kind: resource.Kind}
				gk := gvk.GroupKind()
				if isDouble(gk) {
					continue
				}

				if gk.String() == "Endpoints" { // Services transfer their labels to Endpoints; ignore the latter
					continue
				}

				if _, found := done[gk]; found {
					continue
				}
				done[gk] = struct{}{}

				dc, err := s.dyn.ClientForGroupVersionKind(gvk)
				if err != nil {
					return err
				}

				o, err := dc.Resource(&resource, "").List(metav1.ListOptions{})
				if err != nil {
					return err
				}

				l, ok := o.(*unstructured.UnstructuredList)
				if !ok {
					continue
				}

				for _, i := range l.Items {
					// check that the object is marked by the sync pod
					l := i.GetLabels()
					if l[ownedBySyncPodLabelKey] == "true" {
						// if object is marked, but not in current DB, remove it
						if _, ok := s.db[keyFunc(i.GroupVersionKind().GroupKind(), i.GetNamespace(), i.GetName())]; !ok {
							s.log.Info("Delete " + keyFunc(i.GroupVersionKind().GroupKind(), i.GetNamespace(), i.GetName()))
							err = dc.Resource(&resource, i.GetNamespace()).Delete(i.GetName(), nil)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// contains returns true if haystack contains needle
func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// write synchronises a single object with the API server.
func (s *sync) write(o *unstructured.Unstructured) error {
	dc, err := s.dyn.ClientForGroupVersionKind(o.GroupVersionKind())
	if err != nil {
		return err
	}

	var gr *discovery.APIGroupResources
	for _, g := range s.grs {
		if g.Group.Name == o.GroupVersionKind().Group {
			gr = g
			break
		}
	}
	if gr == nil {
		return errors.New("couldn't find group " + o.GroupVersionKind().Group)
	}

	var res *metav1.APIResource
	for _, r := range gr.VersionedResources[o.GroupVersionKind().Version] {
		if gr.Group.Name == "template.openshift.io" && r.Name == "processedtemplates" {
			continue
		}
		if r.Kind == o.GroupVersionKind().Kind {
			res = &r
			break
		}
	}
	if res == nil {
		return errors.New("couldn't find kind " + o.GroupVersionKind().Kind)
	}

	o = o.DeepCopy() // TODO: do this much earlier

	err = retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		var existing *unstructured.Unstructured
		existing, err = dc.Resource(res, o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			s.log.Info("Create " + keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))
			markSyncPodOwned(o)
			s.fixupRouterArchitecture(o)
			_, err = dc.Resource(res, o.GetNamespace()).Create(o)
			if kerrors.IsAlreadyExists(err) {
				// The "hot path" in write() is Get, check, then maybe Update.
				// Optimising for this has the disadvantage that at cluster
				// creation we can race with API server or controller
				// initialisation. Between Get returning NotFound and us trying
				// to Create, the object might be created.  In this case we
				// return a synthetic Conflict to force a retry.
				err = kerrors.NewConflict(schema.GroupResource{Group: res.Group, Resource: res.Name}, o.GetName(), errors.New("synthetic"))
			}
			return
		}
		if err != nil {
			return
		}

		rv := existing.GetResourceVersion()

		err = clean(*existing)
		if err != nil {
			return
		}
		defaults(*existing)

		s.fixupRouterArchitecture(o)
		markSyncPodOwned(o)

		if !needsUpdate(s.log, existing, o) {
			return
		}
		printDiff(s.log, existing, o)

		o.SetResourceVersion(rv)
		_, err = dc.Resource(res, o.GetNamespace()).Update(o)
		return
	})

	return err
}

func (s *sync) fixupRouterArchitecture(o *unstructured.Unstructured) {
	// TODO: When the old router architecture (type: loadbalancer) is
	// non-existent then move these changes into the yaml files
	if !s.cs.Config.NewRouterArchitecture {
		return
	}

	gk := o.GroupVersionKind().GroupKind()
	if gk.String() == "DaemonSet.apps" && o.GetNamespace() == "default" && o.GetName() == "router" {
		// update with changes
		spec := jsonpath.MustCompile("$.spec.template.spec").MustGetObject(o.Object)
		spec["hostNetwork"] = true
		jsonpath.MustCompile("$.spec.template.spec").Set(o.Object, spec)
	}
	if gk.String() == "Service" && o.GetNamespace() == "default" && o.GetName() == "router" {
		jsonpath.MustCompile("$.spec.type").Set(o.Object, "ClusterIP")
		jsonpath.MustCompile("$.metadata.annotations['service.beta.kubernetes.io/azure-dns-label-name']").Delete(o.Object)
	}
}

// mark object as sync pod owned
func markSyncPodOwned(o *unstructured.Unstructured) {
	l := o.GetLabels()
	if l == nil {
		l = map[string]string{}
	}
	l[ownedBySyncPodLabelKey] = "true"
	o.SetLabels(l)
}

func needsUpdate(log *logrus.Entry, existing, o *unstructured.Unstructured) bool {
	handleSpecialObjects(*existing, *o)

	if reflect.DeepEqual(*existing, *o) {
		return false
	}

	log.Info("Update " + keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))

	return true
}

func printDiff(log *logrus.Entry, existing, o *unstructured.Unstructured) bool {
	// TODO: we should have tests that monitor these diffs:
	// 1) when a cluster is created
	// 2) when sync is run twice back-to-back on the same cluster

	// Don't show a diff if kind is Secret
	gk := o.GroupVersionKind().GroupKind()
	diffShown := false
	if gk.String() != "Secret" {
		for _, diff := range deep.Equal(*existing, *o) {
			log.Info("- " + diff)
			diffShown = true
		}
	}
	return diffShown
}
