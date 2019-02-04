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

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
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

func NewAzureClusterReaderClient(cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	var kc api.Config
	err := latest.Scheme.Convert(cs.Config.AzureClusterReaderKubeconfig, &kc, nil)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(&kc)
}

func NewCustomerReaderClient(cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	kc, err := login("customer-cluster-reader", cs)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}

func NewAdminClient(cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	kc, err := login("admin", cs)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}

func NewCustomerAdminClient(cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	kc, err := login("customer-cluster-admin", cs)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}

func NewEndUserClient(cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	kc, err := login("enduser", cs)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(kc)
}

type ClientSet struct {
	AzureClusterReader *Client
	CustomerReader     *Client
	Admin              *Client
	CustomerAdmin      *Client
	EndUser            *Client
}

// NewClientSet creates a new set of openshift clients scoped for different levels
// of access
func NewClientSet(cs *internalapi.OpenShiftManagedCluster) (*ClientSet, error) {
	c := &ClientSet{}
	var err error
	c.Admin, err = NewAdminClient(cs)
	if err != nil {
		return nil, err
	}
	c.AzureClusterReader, err = NewAzureClusterReaderClient(cs)
	if err != nil {
		return nil, err
	}
	c.CustomerAdmin, err = NewCustomerAdminClient(cs)
	if err != nil {
		return nil, err
	}
	c.CustomerReader, err = NewCustomerReaderClient(cs)
	if err != nil {
		return nil, err
	}
	c.EndUser, err = NewEndUserClient(cs)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func NewDefaultClientSet() (*ClientSet, error) {
	cs, err := shared.DiscoverInternalConfig()
	if err != nil {
		return nil, err
	}
	return NewClientSet(cs)
}
