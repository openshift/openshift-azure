package managedcluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

// ReadConfig returns a config object from a config file
func ReadConfig(path string) (*api.OpenShiftManagedCluster, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, err
	}

	return cs, nil
}

// RestConfigFromV1Config takes a v1 config and returns a kubeconfig
func RestConfigFromV1Config(kc *v1.Config) (*rest.Config, error) {
	var c kapi.Config
	err := latest.Scheme.Convert(kc, &c, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(c, &clientcmd.ConfigOverrides{})
	return kubeconfig.ClientConfig()
}

// WaitForHealthz takes a context, a OpenShiftManagedCluster, and a logrus.Entry
// It waits for the cluster to respond to healthz requests.
func WaitForHealthz(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster) error {
	pool := x509.NewCertPool()
	pool.AddCert(cs.Config.Certificates.Ca.Cert)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: pool,
		},
	}

	// Wait for the healthz to be 200 status
	_, err := wait.ForHTTPStatusOk(ctx, log, t, "https://"+cs.Properties.FQDN+"/healthz")
	return err
}
