package openshift

import (
	oappsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	policyv1beta1client "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

type Client struct {
	AppsV1          appsv1client.AppsV1Interface
	AuthorizationV1 authorizationv1client.AuthorizationV1Interface
	CoreV1          corev1client.CoreV1Interface
	BatchV1         batchv1client.BatchV1Interface
	PolicyV1beta1   policyv1beta1client.PolicyV1beta1Interface
	RbacV1          rbacv1client.RbacV1Interface

	OAppsV1    oappsv1client.AppsV1Interface
	ProjectV1  projectv1client.ProjectV1Interface
	RouteV1    routev1client.RouteV1Interface
	TemplateV1 templatev1client.TemplateV1Interface
	UserV1     userv1client.UserV1Interface
}

func newClientFromRestConfig(config *rest.Config) *Client {
	return &Client{
		AppsV1:          appsv1client.NewForConfigOrDie(config),
		AuthorizationV1: authorizationv1client.NewForConfigOrDie(config),
		CoreV1:          corev1client.NewForConfigOrDie(config),
		PolicyV1beta1:   policyv1beta1client.NewForConfigOrDie(config),
		RbacV1:          rbacv1client.NewForConfigOrDie(config),
		BatchV1:         batchv1client.NewForConfigOrDie(config),

		OAppsV1:    oappsv1client.NewForConfigOrDie(config),
		ProjectV1:  projectv1client.NewForConfigOrDie(config),
		RouteV1:    routev1client.NewForConfigOrDie(config),
		TemplateV1: templatev1client.NewForConfigOrDie(config),
		UserV1:     userv1client.NewForConfigOrDie(config),
	}
}

func newClientFromKubeConfig(kc *api.Config) (*Client, error) {
	kubeconfig := clientcmd.NewDefaultClientConfig(*kc, &clientcmd.ConfigOverrides{})

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return newClientFromRestConfig(restconfig), nil
}

func NewAzureClusterReaderClient() (*Client, error) {
	cs, err := managedcluster.ReadConfig("../../_data/containerservice.yaml")
	if err != nil {
		return nil, err
	}

	var kc api.Config
	err = latest.Scheme.Convert(cs.Config.AzureClusterReaderKubeconfig, &kc, nil)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(&kc)
}

func NewCustomerReaderClient() (*Client, error) {
	kc, err := login("customer-cluster-reader")
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}

func NewAdminClient() (*Client, error) {
	kc, err := login("admin")
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}

func NewCustomerAdminClient() (*Client, error) {
	kc, err := login("customer-cluster-admin")
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}

func NewEndUserClient() (*Client, error) {
	kc, err := login("enduser")
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}
