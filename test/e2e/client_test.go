//+build e2e

package e2e

import (
	"fmt"
	"time"

	project "github.com/openshift/api/project/v1"
	projectclient "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var c *testClient

type testClient struct {
	kc        *kubernetes.Clientset
	pc        *projectclient.ProjectV1Client
	rc        *routev1client.RouteV1Client
	tc        *templatev1client.TemplateV1Client
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

	// create a project client for creating and tearing down namespaces
	pc, err := projectclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create a template client
	tc, err := templatev1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create a route client

	rc, err := routev1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return &testClient{
		kc: kc,
		pc: pc,
		rc: rc,
		tc: tc,
	}
}

func (t *testClient) createProject(namespace string) error {
	if _, err := t.pc.ProjectRequests().Create(&project.ProjectRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}}); err != nil {
		return err
	}
	t.namespace = namespace

	if err := wait.PollImmediate(2*time.Second, time.Minute, t.selfSarSuccess); err != nil {
		return fmt.Errorf("failed to wait for self-sar success: %v", err)
	}
	if err := wait.PollImmediate(2*time.Second, time.Minute, t.defaultServiceAccountIsReady); err != nil {
		return fmt.Errorf("failed to wait for the default service account provision: %v", err)
	}
	return nil
}

func (t *testClient) cleanupProject(timeout time.Duration) error {
	if t.namespace == "" {
		return nil
	}
	if err := t.pc.Projects().Delete(t.namespace, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := wait.PollImmediate(2*time.Second, timeout, t.projectIsCleanedUp); err != nil {
		return fmt.Errorf("failed to wait for project cleanup: %v", err)
	}
	return nil
}
