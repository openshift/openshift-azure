//+build e2e

package e2e

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var c *testClient

type testClient struct {
	kc        *kubernetes.Clientset
	namespace string
}

func newTestClient(kubeconfig string) *testClient {
	var err error
	var config *rest.Config

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
	} else {
		// use in-cluster config if no kubeconfig has been specified
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	// create the clientset
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return &testClient{
		kc: kc,
	}
}

func (t *testClient) createNamespace(namespace string) {
	// TODO: Create a project request
	t.namespace = namespace
	// TODO: Wait for a successful SAR check
}

func (t *testClient) cleanupNamespace(timeout time.Duration) {
	if t.namespace == "" {
		return
	}

	// TODO: Do a project delete and wait for the namespace to cleanup
}
