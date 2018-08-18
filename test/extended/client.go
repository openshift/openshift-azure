package extended

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/checks"

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
	kaggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// Interface exposes the methods a client needs to implement
// for the syncing process of the addons.
type Interface interface {
	UpdateDynamicClient() error
	Create(unstructured.Unstructured) error
	Delete(unstructured.Unstructured) error
	Unmarshal([]byte) (unstructured.Unstructured, error)
	PoolStatus(unstructured.Unstructured) error
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

func newClient() (Interface, error) {

	var kubeconfig string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	} else {
		kubeconfig = *flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	restconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
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
// information and dynamic client po	ol.
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

func (c *client) Create(o unstructured.Unstructured) error {
	err := write(c.dyn, c.grs, &o)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) Delete(o unstructured.Unstructured) error {
	err := delete(c.dyn, c.grs, &o)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) Unmarshal(b []byte) (unstructured.Unstructured, error) {
	// can't use straight yaml.Unmarshal() because it universally mangles yaml
	// integers into float64s, whereas the Kubernetes client library uses int64s
	// wherever it can.  Such a difference can cause us to update objects when
	// we don't actually need to.
	json, err := yaml.YAMLToJSON(b)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	var o unstructured.Unstructured
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(json, nil, &o)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return o, nil
}

// resolveResource resolved resource type and return dynamic client and resource itself
func resolveResource(dyn dynamic.ClientPool, grs []*discovery.APIGroupResources, o *unstructured.Unstructured) (dynamic.Interface, *metav1.APIResource, error) {
	dc, err := dyn.ClientForGroupVersionKind(o.GroupVersionKind())
	if err != nil {
		return nil, nil, err
	}

	var gr *discovery.APIGroupResources
	for _, g := range grs {
		if g.Group.Name == o.GroupVersionKind().Group {
			gr = g
			break
		}
	}
	if gr == nil {
		return nil, nil, errors.New("couldn't find group " + o.GroupVersionKind().Group)
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
		return nil, nil, errors.New("couldn't find kind " + o.GroupVersionKind().Kind)
	}
	return dc, res, nil
}

// write synchronises a single object with the API server.
func write(dyn dynamic.ClientPool, grs []*discovery.APIGroupResources, o *unstructured.Unstructured) error {
	dc, res, err := resolveResource(dyn, grs, o)
	if err != nil {
		return err
	}

	o = o.DeepCopy() // TODO: do this much earlier
	err = retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		log.Infof("Create " + o.GetName())
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
	})
	return err
}

// delete a single object from the API server.
func delete(dyn dynamic.ClientPool, grs []*discovery.APIGroupResources, o *unstructured.Unstructured) error {
	dc, res, err := resolveResource(dyn, grs, o)
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
		log.Infof("Delete " + o.GetName())
		err = dc.Resource(res, o.GetNamespace()).Delete(o.GetName(), &metav1.DeleteOptions{})
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
	})
	return err
}

func (c *client) PoolStatus(o unstructured.Unstructured) error {
	//TODO
	return nil
}
