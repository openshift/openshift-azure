package managedcluster

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	"github.com/openshift/openshift-azure/pkg/api"
)

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

func ClientsetFromConfig(cs *api.OpenShiftManagedCluster) (*kubernetes.Clientset, error) {
	var kc kapi.Config
	err := latest.Scheme.Convert(cs.Config.AdminKubeconfig, &kc, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(kc, &clientcmd.ConfigOverrides{})

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restconfig)
}
