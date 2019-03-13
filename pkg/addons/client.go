package addons

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
	kapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/retry"
	kaggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

// Interface exposes the methods a client needs to implement
// for the syncing process of the addons.
type Interface interface {
	ApplyResources(filter func(unstructured.Unstructured) bool, db map[string]unstructured.Unstructured, keys []string) error
	UpdateDynamicClient() error
	ServiceCatalogExists() (bool, error)
	CRDReady(name string) (bool, error)
	DeleteOrphans(db map[string]unstructured.Unstructured) error
}

// client implements Interface
var _ Interface = &client{}

type client struct {
	restconfig *rest.Config
	ac         *kaggregator.Clientset
	ae         *kapiextensions.Clientset
	cli        *discovery.DiscoveryClient
	dyn        dynamic.ClientPool
	grs        []*discovery.APIGroupResources
	log        *logrus.Entry
}

func newClient(ctx context.Context, log *logrus.Entry, cs *acsapi.OpenShiftManagedCluster, dryRun bool) (Interface, error) {
	if dryRun {
		return &dryClient{}, nil
	}

	restconfig, err := managedcluster.RestConfigFromV1Config(cs.Config.AdminKubeconfig)
	if err != nil {
		return nil, err
	}

	restconfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()

	ac, err := kaggregator.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	ae, err := kapiextensions.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	cli, err := discovery.NewDiscoveryClientForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	c := &client{
		restconfig: restconfig,
		ac:         ac,
		ae:         ae,
		cli:        cli,
		log:        log,
	}
	transport, err := rest.TransportFor(c.restconfig)
	if err != nil {
		return nil, err
	}
	if _, err := wait.ForHTTPStatusOk(ctx, log, transport, c.restconfig.Host+"/healthz"); err != nil {
		return nil, err
	}

	if err := c.UpdateDynamicClient(); err != nil {
		return nil, err
	}

	return c, nil
}

// UpdateDynamicClient updates the client's server API group resource
// information and dynamic client pool.
func (c *client) UpdateDynamicClient() error {
	grs, err := discovery.GetAPIGroupResources(c.cli)
	if err != nil {
		return err
	}
	c.grs = grs

	rm := discovery.NewRESTMapper(c.grs, meta.InterfacesForUnstructured)
	c.dyn = dynamic.NewClientPool(c.restconfig, rm, dynamic.LegacyAPIPathResolverFunc)

	return nil
}

// ApplyResources creates or updates all resources in db that match the provided filter.
func (c *client) ApplyResources(filter func(unstructured.Unstructured) bool, db map[string]unstructured.Unstructured, keys []string) error {
	for _, k := range keys {
		o := db[k]

		if !filter(o) {
			continue
		}

		if err := write(c.log, c.dyn, c.grs, &o); err != nil {
			return err
		}
	}
	return nil
}

// DeleteOrphans looks for the "belongs-to-syncpod: yes" annotation, if found and object not in current db, remove it.
func (c *client) DeleteOrphans(db map[string]unstructured.Unstructured) error {
	c.log.Info("Deleting Orphan Objects from the running cluster")
	for _, gr := range c.grs {
		gv, err := schema.ParseGroupVersion(gr.Group.PreferredVersion.GroupVersion)
		if err != nil {
			return err
		}

		for _, resource := range gr.VersionedResources[gr.Group.PreferredVersion.Version] {
			if strings.ContainsRune(resource.Name, '/') { // no subresources
				continue
			}

			if !contains(resource.Verbs, "list") {
				continue
			}

			gvk := gv.WithKind(resource.Kind)
			gk := gvk.GroupKind()
			if IsDouble(gk) {
				continue
			}

			if gk.String() == "Endpoints" { // Services transfer their labels to Endpoints; ignore the latter
				continue
			}

			dc, err := c.dyn.ClientForGroupVersionKind(gvk)
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
					if _, ok := db[KeyFunc(i.GroupVersionKind().GroupKind(), i.GetNamespace(), i.GetName())]; !ok {
						c.log.Info("Delete " + KeyFunc(i.GroupVersionKind().GroupKind(), i.GetNamespace(), i.GetName()))
						err = dc.Resource(&resource, i.GetNamespace()).Delete(i.GetName(), nil)
						if err != nil {
							return err
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
func write(log *logrus.Entry, dyn dynamic.ClientPool, grs []*discovery.APIGroupResources, o *unstructured.Unstructured) error {
	dc, err := dyn.ClientForGroupVersionKind(o.GroupVersionKind())
	if err != nil {
		return err
	}

	var gr *discovery.APIGroupResources
	for _, g := range grs {
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
			log.Info("Create " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))
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

		err = Clean(*existing)
		if err != nil {
			return
		}
		Default(*existing)

		markSyncPodOwned(o)

		if !needsUpdate(log, existing, o) {
			return
		}
		printDiff(log, existing, o)

		o.SetResourceVersion(rv)
		_, err = dc.Resource(res, o.GetNamespace()).Update(o)
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

func needsUpdate(log *logrus.Entry, existing, o *unstructured.Unstructured) bool {
	handleSpecialObjects(*existing, *o)

	if reflect.DeepEqual(*existing, *o) {
		log.Info("Skip " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))
		return false
	}

	log.Info("Update " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))

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

// ServiceCatalogExists returns whether the service catalog API exists.
func (c *client) ServiceCatalogExists() (bool, error) {
	return ready.APIServiceIsReady(c.ac.ApiregistrationV1().APIServices(), "v1beta1.servicecatalog.k8s.io")()
}

// CRDReady returns whether the required CRDs got registered.
func (c *client) CRDReady(name string) (bool, error) {
	return ready.CRDReady(c.ae.ApiextensionsV1beta1().CustomResourceDefinitions(), name)()
}

type dryClient struct{}

// dryClient implements Interface
var _ Interface = &dryClient{}

func (c *dryClient) ApplyResources(filter func(unstructured.Unstructured) bool, db map[string]unstructured.Unstructured, keys []string) error {
	return nil
}
func (c *dryClient) UpdateDynamicClient() error                                  { return nil }
func (c *dryClient) ServiceCatalogExists() (bool, error)                         { return true, nil }
func (c *dryClient) CRDReady(name string) (bool, error)                          { return true, nil }
func (c *dryClient) DeleteOrphans(db map[string]unstructured.Unstructured) error { return nil }
