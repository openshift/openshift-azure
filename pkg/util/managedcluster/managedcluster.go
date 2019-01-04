package managedcluster

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/api"
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
