package network

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// VirtualNetworksPeeringsClient is a minimal interface for azure NewVirtualNetworksPeeringsClient
type VirtualNetworksPeeringsClient interface {
	Delete(ctx context.Context, resourceGroupName string, virtualNetworkName string, instanceID string) (network.VirtualNetworkPeeringsDeleteFuture, error)
	List(ctx context.Context, resourceGroupName string, virtualNetworkName string) (network.VirtualNetworkPeeringListResultPage, error)
	azureclient.Client
}

type virtualNetworkPeeringsClient struct {
	network.VirtualNetworkPeeringsClient
}

var _ VirtualNetworksPeeringsClient = &virtualNetworkPeeringsClient{}

// NewVirtualNetworksPeeringsClient creates a new VirtualMachineScaleSetVMsClient
func NewVirtualNetworksPeeringsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VirtualNetworksPeeringsClient {
	client := network.NewVirtualNetworkPeeringsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.VirtualNetworkPeeringsClient", &client.Client, authorizer)

	return &virtualNetworkPeeringsClient{
		VirtualNetworkPeeringsClient: client,
	}
}

func (c *virtualNetworkPeeringsClient) List(ctx context.Context, resourceGroupName string, virtualNetworkName string) (network.VirtualNetworkPeeringListResultPage, error) {
	nwRes, err := c.VirtualNetworkPeeringsClient.List(ctx, resourceGroupName, virtualNetworkName)
	return nwRes, err
}

func (c *virtualNetworkPeeringsClient) Client() autorest.Client {
	return c.VirtualNetworkPeeringsClient.Client
}
