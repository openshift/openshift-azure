package healthcheck

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-05-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/checks"
)

type azureNetworkClient struct {
	lb  network.LoadBalancerFrontendIPConfigurationsClient
	eip network.PublicIPAddressesClient
}

func newAzureClients(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) (*azureNetworkClient, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}
	lbc := network.NewLoadBalancerFrontendIPConfigurationsClient(cs.Properties.AzProfile.SubscriptionID)
	ipc := network.NewPublicIPAddressesClient(cs.Properties.AzProfile.SubscriptionID)
	lbc.Authorizer = authorizer
	ipc.Authorizer = authorizer

	ac := azureNetworkClient{
		eip: ipc,
		lb:  lbc,
	}

	return &ac, nil
}

func newKubernetesClientset(ctx context.Context, config *v1.Config) (*kubernetes.Clientset, error) {
	kubeconfig, err := getKubeconfigFromV1Config(config)
	if err != nil {
		return nil, err
	}

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	t, err := rest.TransportFor(restconfig)
	if err != nil {
		return nil, err
	}

	// Wait for the healthz to be 200 status
	err = checks.WaitForHTTPStatusOk(ctx, t, restconfig.Host+"/healthz")
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restconfig)
}

// GetKubeconfigFromV1Config takes a v1 config and returns a kubeconfig
func getKubeconfigFromV1Config(kc *v1.Config) (clientcmd.ClientConfig, error) {
	var c api.Config

	if err := latest.Scheme.Convert(kc, &c, nil); err != nil {
		return nil, err
	}

	return clientcmd.NewDefaultClientConfig(c, &clientcmd.ConfigOverrides{}), nil
}
