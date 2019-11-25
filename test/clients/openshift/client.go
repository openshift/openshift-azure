package openshift

import (
	servicecatalogv1beta1client "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
	oappsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	networkv1client "github.com/openshift/client-go/network/clientset/versioned/typed/network/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	securityv1client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	policyv1beta1client "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
)

type Client struct {
	config                *rest.Config
	Discovery             discovery.DiscoveryInterface
	AppsV1                appsv1client.AppsV1Interface
	AuthorizationV1       authorizationv1client.AuthorizationV1Interface
	CoreV1                corev1client.CoreV1Interface
	BatchV1               batchv1client.BatchV1Interface
	NetworkV1             networkv1client.NetworkV1Interface
	PolicyV1beta1         policyv1beta1client.PolicyV1beta1Interface
	RbacV1                rbacv1client.RbacV1Interface
	ServicecatalogV1beta1 servicecatalogv1beta1client.ServicecatalogV1beta1Interface

	OAppsV1    oappsv1client.AppsV1Interface
	BuildV1    buildv1client.BuildV1Interface
	ProjectV1  projectv1client.ProjectV1Interface
	RouteV1    routev1client.RouteV1Interface
	SecurityV1 securityv1client.SecurityV1Interface
	TemplateV1 templatev1client.TemplateV1Interface
	UserV1     userv1client.UserV1Interface
}

func newClientFromRestConfig(config *rest.Config) *Client {
	return &Client{
		config:                config,
		Discovery:             discovery.NewDiscoveryClientForConfigOrDie(config),
		AppsV1:                appsv1client.NewForConfigOrDie(config),
		AuthorizationV1:       authorizationv1client.NewForConfigOrDie(config),
		CoreV1:                corev1client.NewForConfigOrDie(config),
		NetworkV1:             networkv1client.NewForConfigOrDie(config),
		PolicyV1beta1:         policyv1beta1client.NewForConfigOrDie(config),
		RbacV1:                rbacv1client.NewForConfigOrDie(config),
		BatchV1:               batchv1client.NewForConfigOrDie(config),
		ServicecatalogV1beta1: servicecatalogv1beta1client.NewForConfigOrDie(config),

		OAppsV1:    oappsv1client.NewForConfigOrDie(config),
		BuildV1:    buildv1client.NewForConfigOrDie(config),
		ProjectV1:  projectv1client.NewForConfigOrDie(config),
		RouteV1:    routev1client.NewForConfigOrDie(config),
		SecurityV1: securityv1client.NewForConfigOrDie(config),
		TemplateV1: templatev1client.NewForConfigOrDie(config),
		UserV1:     userv1client.NewForConfigOrDie(config),
	}
}

func newClientFromKubeConfig(log *logrus.Entry, cs *internalapi.OpenShiftManagedCluster, kc *v1.Config) (*Client, error) {
	restconfig, err := kubeclient.NewRestConfig(log, kc, cs, true)
	if err != nil {
		return nil, err
	}

	return newClientFromRestConfig(restconfig), nil
}

func NewAdminClient(log *logrus.Entry, cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	kc, err := login("admin", cs)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(log, cs, kc)
}

func NewCustomerAdminClient(log *logrus.Entry, cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	kc, err := login("customer-cluster-admin", cs)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(log, cs, kc)
}

func NewEndUserClient(log *logrus.Entry, cs *internalapi.OpenShiftManagedCluster) (*Client, error) {
	kc, err := login("enduser", cs)
	if err != nil {
		return nil, err
	}

	return newClientFromKubeConfig(log, cs, kc)
}

type ClientSet struct {
	Admin         *Client
	CustomerAdmin *Client
	EndUser       *Client
}

// NewClientSet creates a new set of openshift clients scoped for different levels
// of access
func NewClientSet(log *logrus.Entry, cs *internalapi.OpenShiftManagedCluster) (*ClientSet, error) {
	c := &ClientSet{}
	var err error
	c.Admin, err = NewAdminClient(log, cs)
	if err != nil {
		return nil, err
	}
	c.CustomerAdmin, err = NewCustomerAdminClient(log, cs)
	if err != nil {
		return nil, err
	}
	c.EndUser, err = NewEndUserClient(log, cs)
	if err != nil {
		return nil, err
	}
	return c, nil
}
