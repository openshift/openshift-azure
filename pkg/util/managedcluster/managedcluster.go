package managedcluster

import (
	"context"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
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

// getKubeconfigFromV1Config takes a v1 config and returns a kubeconfig
func getRestConfigFromV1Config(kc *v1.Config) (*rest.Config, error) {
	var c kapi.Config
	err := latest.Scheme.Convert(kc, &c, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(c, &clientcmd.ConfigOverrides{})
	return kubeconfig.ClientConfig()
}

// ClientsetFromV1Config takes a v1 config and returns a Clientset
func ClientsetFromV1Config(config *v1.Config) (*kubernetes.Clientset, error) {
	restconfig, err := getRestConfigFromV1Config(config)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restconfig)
}

// WaitForHealthz takes a context, v1 config.
// It waits for the cluster to respond to healthz requests.
func WaitForHealthz(ctx context.Context, config *v1.Config, log *logrus.Entry) error {
	restconfig, err := getRestConfigFromV1Config(config)
	if err != nil {
		return err
	}

	t, err := rest.TransportFor(restconfig)
	if err != nil {
		return err
	}

	// Wait for the healthz to be 200 status
	_, err = wait.ForHTTPStatusOk(ctx, t, restconfig.Host+"/healthz", log)
	return err
}
