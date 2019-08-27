package network

import (
	"context"

	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	pl "github.com/openshift/openshift-azure/pkg/util/azureclient/network/privatelink"
)

// PrivateEndpointsClient is a minimal interface for azure NewPrivateEndpointsClient
type PrivateEndpointsClient interface {
	Get(ctx context.Context, resourceGroupName string, privateEndpointName string, expand string) (result pl.PrivateEndpoint, err error)
	Delete(ctx context.Context, resourceGroupName string, privateEndpointName string) (result pl.PrivateEndpointsDeleteFuture, err error)
}

type privateEndpointsClient struct {
	pl.PrivateEndpointsClient
}

var _ PrivateEndpointsClient = &privateEndpointsClient{}

// NewPrivateEndpointsClient creates a new PrivateEndpointsClient
func NewPrivateEndpointsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) PrivateEndpointsClient {
	client := pl.NewPrivateEndpointsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.PublicIPAddressesClient", &client.Client, authorizer)

	return &privateEndpointsClient{
		PrivateEndpointsClient: client,
	}
}

// PrivateLinkServicesClient is a minimal interface for azure NewNewPrivateLinkServicesClient
type PrivateLinkServicesClient interface {
	Get(ctx context.Context, resourceGroupName string, serviceName string, expand string) (result pl.PrivateLinkService, err error)
}

type privateLinkServicesClient struct {
	pl.PrivateLinkServicesClient
}

var _ PrivateLinkServicesClient = &privateLinkServicesClient{}

// NewPrivateLinkServicesClient creates a new NewPrivateLinkServicesClient
func NewPrivateLinkServicesClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) PrivateLinkServicesClient {
	client := pl.NewPrivateLinkServicesClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.PublicIPAddressesClient", &client.Client, authorizer)

	return &privateLinkServicesClient{
		PrivateLinkServicesClient: client,
	}
}
