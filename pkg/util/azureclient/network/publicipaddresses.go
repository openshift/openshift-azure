package network

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// PublicIPAddressesClient is a minimal interface for azure NewPublicIPAddressesClient
type PublicIPAddressesClient interface {
	Get(ctx context.Context, resourceGroupName string, publicIPAddressName string, expand string) (network.PublicIPAddress, error)
	ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx context.Context, resourceGroupName string, scaleSetName string) (network.PublicIPAddressListResultIterator, error)
}

type publicIPAddressesClient struct {
	network.PublicIPAddressesClient
}

var _ PublicIPAddressesClient = &publicIPAddressesClient{}

// NewPublicIPAddressesClient creates a new PublicIPAddressesClient
func NewPublicIPAddressesClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) PublicIPAddressesClient {
	client := network.NewPublicIPAddressesClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.PublicIPAddressesClient", &client.Client, authorizer)

	return &publicIPAddressesClient{
		PublicIPAddressesClient: client,
	}
}
