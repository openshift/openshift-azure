package addons

import (
	"context"
	"errors"
	"log"
	"reflect"

	"github.com/openshift/openshift-azure/pkg/checks"

	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/retry"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	kaggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// Interface exposes the methods a client needs to implement
// for the syncing process of the addons.
type Interface interface {
	ApplyResources(filter func(unstructured.Unstructured) bool, db map[string]unstructured.Unstructured, keys []string) error
	UpdateDynamicClient() error
	ServiceCatalogExists() (bool, error)
}

// client implements Interface
var _ Interface = &client{}

type client struct {
	restconfig *rest.Config
	ac         *kaggregator.Clientset
	cli        *discovery.DiscoveryClient
	dyn        dynamic.ClientPool
	grs        []*discovery.APIGroupResources
}

func newClient(dryRun bool) (Interface, error) {
	if dryRun {
		return &dryClient{}, nil
	}

	restconfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}
	restconfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()

	ac, err := kaggregator.NewForConfig(restconfig)
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
		cli:        cli,
	}

	transport, err := rest.TransportFor(c.restconfig)
	if err != nil {
		return nil, err
	}

	if err := checks.WaitForHTTPStatusOk(context.Background(), transport, c.restconfig.Host+"/healthz"); err != nil {
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

		if err := write(c.dyn, c.grs, &o); err != nil {
			return err
		}
	}
	return nil
}

// write synchronises a single object with the API server.
func write(dyn dynamic.ClientPool, grs []*discovery.APIGroupResources, o *unstructured.Unstructured) error {
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
			log.Println("Create " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))
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

		if reflect.DeepEqual(*existing, *o) {
			log.Println("Skip " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))
			return
		}
		// this part of code should be executed only for updates
		// TODO: Split flows to separate functions
		handleSpecialObjects(*existing, *o)

		log.Println("Update " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))

		// TODO: we should have tests that monitor these diffs:
		// 1) when a cluster is created
		// 2) when sync is run twice back-to-back on the same cluster
		for _, diff := range equality.Semantic.DeepEqual(*existing, *o) {
			log.Println("- " + diff)
		}

		o.SetResourceVersion(rv)
		_, err = dc.Resource(res, o.GetNamespace()).Update(o)
		return
	})

	return err
}

// ServiceCatalogExists returns whether the service catalog API exists.
func (c *client) ServiceCatalogExists() (bool, error) {
	svc, err := c.ac.ApiregistrationV1().APIServices().Get("v1beta1.servicecatalog.k8s.io", metav1.GetOptions{})
	switch {
	case kerrors.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, err
	}
	for _, cond := range svc.Status.Conditions {
		if cond.Type == apiregistrationv1.Available &&
			cond.Status == apiregistrationv1.ConditionTrue {
			return true, nil
		}
	}
	return false, nil
}

type dryClient struct{}

// dryClient implements Interface
var _ Interface = &dryClient{}

func (c *dryClient) ApplyResources(filter func(unstructured.Unstructured) bool, db map[string]unstructured.Unstructured, keys []string) error {
	return nil
}
func (c *dryClient) UpdateDynamicClient() error          { return nil }
func (c *dryClient) ServiceCatalogExists() (bool, error) { return true, nil }
