//+build e2e

package e2e

import (
	"time"

	project "github.com/openshift/api/project/v1"
	projectclient "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var c *testClient

type testClient struct {
	kc        *kubernetes.Clientset
	pc        *projectclient.ProjectV1Client
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

	pc, err := projectclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return &testClient{
		kc: kc,
		pc: pc,
	}
}

func (t *testClient) createNamespace(namespace string) error {
	if _, err := t.pc.ProjectRequests().Create(&project.ProjectRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}}); err != nil {
		return err
	}
	t.namespace = namespace
	// TODO: Wait for a successful SAR check
	return nil
}

func (t *testClient) cleanupNamespace(timeout time.Duration) error {
	if t.namespace == "" {
		return nil
	}

	if err := t.pc.Projects().Delete(t.namespace, &metav1.DeleteOptions{}); err != nil {
		return err
	}

	// TODO: Wait for the namespace to cleanup
	return nil
}
