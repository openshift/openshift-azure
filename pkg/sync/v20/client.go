package sync

import (
	"errors"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamic "k8s.io/client-go/deprecated-dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/util/retry"

	"github.com/openshift/openshift-azure/pkg/util/cmp"
)

// updateDynamicClient updates the client's server API group resource
// information and dynamic client pool.
func (s *sync) updateDynamicClient() error {
	grs, err := restmapper.GetAPIGroupResources(s.cli)
	if err != nil {
		return err
	}
	s.grs = grs

	rm := restmapper.NewDiscoveryRESTMapper(s.grs)
	s.dyn = dynamic.NewClientPool(s.restconfig, rm, dynamic.LegacyAPIPathResolverFunc)

	return nil
}

// applyResources creates or updates all resources in db that match the provided
// filter.
func (s *sync) applyResources(filter func(unstructured.Unstructured) bool, keys []string) error {
	for _, k := range keys {
		o := s.db[k]

		if !filter(o) || strings.ToLower(o.GetLabels()[optionallyApplySyncPodLabelKey]) == "false" {
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
						// if object is marked as owned by the sync pod
						// but not in current DB, OR marked as don't apply
						// then remove it
						entry, inDB := s.db[keyFunc(i.GroupVersionKind().GroupKind(), i.GetNamespace(), i.GetName())]
						optionallyApply := (entry.GetLabels()[optionallyApplySyncPodLabelKey] != "false")
						if !inDB || !optionallyApply {
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

	var gr *restmapper.APIGroupResources
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

		markSyncPodOwned(o)

		if !s.needsUpdate(existing, o) {
			return
		}
		printDiff(s.log, existing, o)

		o.SetResourceVersion(rv)
		_, err = dc.Resource(res, o.GetNamespace()).Update(o)
		if err != nil {
			if strings.Contains(err.Error(), "updates to parameters are forbidden") ||
				(strings.Contains(err.Error(), "field is immutable") && strings.Contains(o.GetName(), "omsagent")) {
				s.log.Infof("object %s is not updateable, will delete and re-create", o.GetName())
				err = dc.Resource(res, o.GetNamespace()).Delete(o.GetName(), &metav1.DeleteOptions{})
				if err != nil {
					return
				}
				o.SetResourceVersion("")
				_, err = dc.Resource(res, o.GetNamespace()).Create(o)
			}
		}

		return
	})

	return err
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

func (s *sync) needsUpdate(existing, o *unstructured.Unstructured) bool {
	handleSpecialObjects(*existing, *o)

	if reflect.DeepEqual(*existing, *o) {
		return false
	}

	// check if object is marked for reconcile exclusion
	if s.isReconcileProtected(existing, o) {
		return false
	}

	s.log.Info("Update " + keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))

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
		if diff := cmp.Diff(*existing, *o); diff != "" {
			log.Info(diff)
			diffShown = true
		}
	}
	return diffShown
}

// isReconcileProtected check if object is marked with reconcile protection annotation
// this will remove certain objects from being updated post cluster creation
// access to modify these objects is limited by RBAC
// list of the objects is whitelisted only
func (s *sync) isReconcileProtected(existing, o *unstructured.Unstructured) bool {
	// if openshift namespace has annotation so we skip all
	// templates and image-streams in the namespace
	if o.GetNamespace() == "openshift" && s.managedSharedResources == false &&
		(o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "image.openshift.io", Kind: "ImageStream"} ||
			o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "template.openshift.io", Kind: "Template"}) {
		return true
	}

	if o.GetName() == "self-provisioners" &&
		(o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "rbac.authorization.k8s.io", Kind: "ClusterRoleBinding"}) {
		if strings.ToLower(existing.GetAnnotations()[reconcileProtectAnnotationKey]) == "true" {
			return true
		}

		// it turns out that not all annotations are synced from
		// ClusterRoleBinding{,.authorization.openshift.io} (which is what most
		// users modify) to ClusterRoleBinding.rbac.authorization.k8s.io (which
		// is what we work with here).  So, check to see if there is a
		// openshift.io/reconcile-protect: true annotation on the
		// ClusterRoleBinding{,.authorization.openshift.io}.  If so, interpret
		// it as being valid for the
		// ClusterRoleBinding.rbac.authorization.k8s.io.
		crb, err := s.auth.ClusterRoleBindings().Get(o.GetName(), metav1.GetOptions{})
		if err == nil && strings.ToLower(crb.Annotations[reconcileProtectAnnotationKey]) == "true" {
			return true
		}
	}

	if strings.ToLower(existing.GetAnnotations()[reconcileProtectAnnotationKey]) == "true" {
		// openshift namespace for shared-resources
		if (o.GetName() == "openshift" &&
			o.GroupVersionKind().GroupKind() == schema.GroupKind{Kind: "Namespace"}) {
			return true
		}
		// individual shared-resources object in openshift namespace
		if o.GetNamespace() == "openshift" &&
			(o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "image.openshift.io", Kind: "ImageStream"} ||
				o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "template.openshift.io", Kind: "Template"}) {
			return true
		}
		// log analytics agent data collection configuration
		if o.GetNamespace() == "openshift-azure-logging" &&
			(o.GroupVersionKind().GroupKind() == schema.GroupKind{Kind: "ConfigMap"} &&
				o.GetName() == "container-azm-ms-agentconfig") {
			return true
		}
		// SecurityContextConstraints
		if (o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "security.openshift.io", Kind: "SecurityContextConstraints"}) {
			return true
		}
	}

	return false
}
