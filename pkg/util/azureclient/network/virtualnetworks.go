package network

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE VirtualNetworksClient,VirtualNetworksPeeringsClient,PublicIPAddressesClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworkClient
type VirtualNetworksClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, expand string) (result network.VirtualNetwork, err error)
	List(ctx context.Context, resourceGroupName string) (result network.VirtualNetworkListResultPage, err error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters network.VirtualNetwork) (network.VirtualNetworksCreateOrUpdateFuture, error)
	Delete(ctx context.Context, resourceGroupName string, virtualNetworkName string) (network.VirtualNetworksDeleteFuture, error)
	azureclient.Client
}

type virtualNetworksClient struct {
	network.VirtualNetworksClient
}

var _ VirtualNetworksClient = &virtualNetworksClient{}

// NewVirtualNetworkClient creates a new VirtualNetworkClient
func NewVirtualNetworkClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VirtualNetworksClient {
	client := network.NewVirtualNetworksClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.VirtualNetworksClient", &client.Client, authorizer)

	return &virtualNetworksClient{
		VirtualNetworksClient: client,
	}
}

func (c *virtualNetworksClient) Client() autorest.Client {
	return c.VirtualNetworksClient.Client
}
