package healthcheck

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-05-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

type azureNetworkClient struct {
	lb  network.LoadBalancerFrontendIPConfigurationsClient
	eip network.PublicIPAddressesClient
}

func newAzureClients(ctx context.Context, cs *acsapi.ContainerService) (*azureNetworkClient, error) {

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
