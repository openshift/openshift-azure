package network

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// PrivateEndpointsClient is a minimal interface for azure NewPrivateEndpointsClient
type PrivateEndpointsClient interface {
	Get(ctx context.Context, resourceGroupName string, privateEndpointName string, expand string) (result network.PrivateEndpoint, err error)
	Delete(ctx context.Context, resourceGroupName string, privateEndpointName string) (result network.PrivateEndpointsDeleteFuture, err error)
}

type privateEndpointsClient struct {
	network.PrivateEndpointsClient
}

var _ PrivateEndpointsClient = &privateEndpointsClient{}

// NewPrivateEndpointsClient creates a new PrivateEndpointsClient
func NewPrivateEndpointsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) PrivateEndpointsClient {
	client := network.NewPrivateEndpointsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.PublicIPAddressesClient", &client.Client, authorizer)

	return &privateEndpointsClient{
		PrivateEndpointsClient: client,
	}
}

// PrivateLinkServicesClient is a minimal interface for azure NewNewPrivateLinkServicesClient
type PrivateLinkServicesClient interface {
	Get(ctx context.Context, resourceGroupName string, serviceName string, expand string) (result network.PrivateLinkService, err error)
}

type privateLinkServicesClient struct {
	network.PrivateLinkServicesClient
}

var _ PrivateLinkServicesClient = &privateLinkServicesClient{}

// NewPrivateLinkServicesClient creates a new NewPrivateLinkServicesClient
func NewPrivateLinkServicesClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) PrivateLinkServicesClient {
	client := network.NewPrivateLinkServicesClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.PublicIPAddressesClient", &client.Client, authorizer)

	return &privateLinkServicesClient{
		PrivateLinkServicesClient: client,
	}
}
