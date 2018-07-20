package addons

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/go-test/deep"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/retry"
	kaggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type client struct {
	restconfig *rest.Config
	ac         *kaggregator.Clientset
	cli        *discovery.DiscoveryClient
	dyn        dynamic.ClientPool
	grs        []*discovery.APIGroupResources

	// TODO: Instead of plumbing dryRun use an interface
	// and separate client implementations (one for prod
	// one for dryRun, and potentially one for tests).
	dryRun bool
}

func newClient(dryRun bool) (*client, error) {
	if dryRun {
		return &client{dryRun: true}, nil
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

	if err := c.updateDynamicClient(); err != nil {
		return nil, err
	}

	return c, nil
}

// updateDynamicClient updates the client's server API group resource
// information and dynamic client pool.
func (c *client) updateDynamicClient() error {
	grs, err := discovery.GetAPIGroupResources(c.cli)
	if err != nil {
		return err
	}
	c.grs = grs

	rm := discovery.NewRESTMapper(c.grs, meta.InterfacesForUnstructured)
	c.dyn = dynamic.NewClientPool(c.restconfig, rm, dynamic.LegacyAPIPathResolverFunc)

	return nil
}

func (c *client) waitForHealthz() error {
	if c.dryRun {
		return nil
	}

	transport, err := rest.TransportFor(c.restconfig)
	if err != nil {
		return err
	}

	cli := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequest("GET", c.restconfig.Host+"/healthz", nil)
	if err != nil {
		return err
	}

	for {
		resp, err := cli.Do(req)
		if err, ok := err.(*url.Error); ok && (err.Timeout() || err.Err == io.EOF || err.Err == io.ErrUnexpectedEOF) {
			time.Sleep(time.Second)
			continue
		}
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusOK {
			return nil
		}

		time.Sleep(time.Second)
	}
}

// createResources creates all resources in db that match the provided filter.
func (c *client) createResources(filter func(unstructured.Unstructured) bool, db map[string]unstructured.Unstructured, keys []string) error {
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
			return
		}
		if err != nil {
			return
		}

		rv := existing.GetResourceVersion()
		handleSpecialObjects(*existing, *o)
		err = Clean(*existing)
		if err != nil {
			return
		}
		Default(*existing)

		if reflect.DeepEqual(*existing, *o) {
			log.Println("Skip " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))
			return
		}

		log.Println("Update " + KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()))

		// TODO: we should have tests that monitor these diffs:
		// 1) when a cluster is created
		// 2) when sync is run twice back-to-back on the same cluster
		for _, diff := range deep.Equal(*existing, *o) {
			log.Println("- " + diff)
		}

		o.SetResourceVersion(rv)
		_, err = dc.Resource(res, o.GetNamespace()).Update(o)
		return
	})

	return err
}
