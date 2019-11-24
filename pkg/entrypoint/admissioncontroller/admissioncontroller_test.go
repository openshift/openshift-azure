package admissioncontroller

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"

	"github.com/ghodss/yaml"
	"github.com/openshift/origin/pkg/security/apis/security"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/apis/admission"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/plugin"
	utiltls "github.com/openshift/openshift-azure/pkg/util/tls"
)

var (
	cs             *api.OpenShiftManagedCluster
	client         *fake.Clientset
	imageWhitelist []*regexp.Regexp
	sccs           map[string]*security.SecurityContextConstraints
)

type testAddr struct{}

func (testAddr) Network() string { return "" }
func (testAddr) String() string  { return "" }

type testListener struct {
	c      chan net.Conn
	closed bool
}

func newTestListener() *testListener {
	return &testListener{
		c: make(chan net.Conn),
	}
}

func (l *testListener) Accept() (net.Conn, error) {
	c, ok := <-l.c
	if !ok {
		return nil, fmt.Errorf("listener closed")
	}
	return c, nil
}

func (l *testListener) Close() error {
	if !l.closed {
		close(l.c)
		l.closed = true
	}
	return nil
}

func (*testListener) Addr() net.Addr {
	return testAddr{}
}

func (l *testListener) Dial(network, addr string) (net.Conn, error) {
	c1, c2 := net.Pipe()
	l.c <- c1
	return c2, nil
}

func init() {
	b, err := ioutil.ReadFile("../../../pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		panic(err)
	}

	var config *plugin.Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		panic(err)
	}

	cs = &api.OpenShiftManagedCluster{
		Config: api.Config{
			PluginVersion: config.PluginVersion,
		},
	}

	cs.Config.Certificates.Ca.Key, cs.Config.Certificates.Ca.Cert, err = utiltls.NewCA("ca")
	if err != nil {
		panic(err)
	}

	cs.Config.Certificates.AroAdmissionController.Key, cs.Config.Certificates.AroAdmissionController.Cert, err = utiltls.NewCert(&utiltls.CertParams{
		Subject:     pkix.Name{CommonName: "server"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		SigningKey:  cs.Config.Certificates.Ca.Key,
		SigningCert: cs.Config.Certificates.Ca.Cert,
	})
	if err != nil {
		panic(err)
	}

	cs.Config.Certificates.AroAdmissionControllerClient.Key, cs.Config.Certificates.AroAdmissionControllerClient.Cert, err = utiltls.NewCert(&utiltls.CertParams{
		Subject:     pkix.Name{CommonName: "client"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		SigningKey:  cs.Config.Certificates.Ca.Key,
		SigningCert: cs.Config.Certificates.Ca.Cert,
	})
	if err != nil {
		panic(err)
	}

	client = &fake.Clientset{
		Fake: ktesting.Fake{},
	}
	client.AddReactor("get", "namespaces", func(a ktesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &core.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: a.(ktesting.GetAction).GetName(),
				Annotations: map[string]string{
					"openshift.io/sa.scc.mcs":       "s0:c0,c0",
					"openshift.io/sa.scc.uid-range": "1/10",
				},
			},
		}, nil
	})

	sccs, err = readSyncPodSCCs(cs)
	if err != nil {
		panic(err)
	}
}

func doRequest(c *http.Client, url string, review *admission.AdmissionReview) (*admission.AdmissionReview, error) {
	var err error
	review.Request.Object, err = legacyscheme.Scheme.ConvertToVersion(review.Request.Object, schema.GroupVersions(legacyscheme.Scheme.PrioritizedVersionsAllGroups()))
	if err != nil {
		return nil, err
	}

	gvk := review.Request.Object.GetObjectKind().GroupVersionKind()
	review.Request.Kind.Group = gvk.Group
	review.Request.Kind.Version = gvk.Version
	review.Request.Kind.Kind = gvk.Kind

	buf := &bytes.Buffer{}
	err = codec.Encode(review, buf)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	review = &admission.AdmissionReview{}
	_, _, err = codec.Decode(b, nil, review)
	if err != nil {
		return nil, err
	}

	return review, nil
}
