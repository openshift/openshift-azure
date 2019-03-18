package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"
)

// VirtualNetworksClient is a minimal interface for azure VirtualNetworkClient
type VirtualNetworksClient interface {
	Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, expand string) (result network.VirtualNetwork, err error)
	List(ctx context.Context, resourceGroupName string) (result network.VirtualNetworkListResultPage, err error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters network.VirtualNetwork) (network.VirtualNetworksCreateOrUpdateFuture, error)
	Delete(ctx context.Context, resourceGroupName string, virtualNetworkName string) (network.VirtualNetworksDeleteFuture, error)
	Client
}

type virtualNetworksClient struct {
	network.VirtualNetworksClient
}

var _ VirtualNetworksClient = &virtualNetworksClient{}

// NewVirtualNetworkClient creates a new VirtualNetworkClient
func NewVirtualNetworkClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) VirtualNetworksClient {
	client := network.NewVirtualNetworksClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

	return &virtualNetworksClient{
		VirtualNetworksClient: client,
	}
}

func (c *virtualNetworksClient) Client() autorest.Client {
	return c.VirtualNetworksClient.Client
}

// VirtualNetworksPeeringsClient is a minimal interface for azure NewVirtualNetworksPeeringsClient
type VirtualNetworksPeeringsClient interface {
	Delete(ctx context.Context, resourceGroupName string, virtualNetworkName string, instanceID string) (network.VirtualNetworkPeeringsDeleteFuture, error)
	List(ctx context.Context, resourceGroupName string, virtualNetworkName string) (network.VirtualNetworkPeeringListResultPage, error)
	Client
}

type virtualNetworkPeeringsClient struct {
	network.VirtualNetworkPeeringsClient
}

var _ VirtualNetworksPeeringsClient = &virtualNetworkPeeringsClient{}

// NewVirtualNetworksPeeringsClient creates a new VirtualMachineScaleSetVMsClient
func NewVirtualNetworksPeeringsClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) VirtualNetworksPeeringsClient {
	client := network.NewVirtualNetworkPeeringsClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

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
func NewPublicIPAddressesClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) PublicIPAddressesClient {
	client := network.NewPublicIPAddressesClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

	return &publicIPAddressesClient{
		PublicIPAddressesClient: client,
	}
}

func (c *publicIPAddressesClient) ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx context.Context, resourceGroupName string, scaleSetName string) (network.PublicIPAddressListResultIterator, error) {
	return c.PublicIPAddressesClient.ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx, resourceGroupName, scaleSetName)
}

func (c *publicIPAddressesClient) Get(ctx context.Context, resourceGroupName string, publicIPAddressName string, expand string) (network.PublicIPAddress, error) {
	return c.PublicIPAddressesClient.Get(ctx, resourceGroupName, publicIPAddressName, expand)
}
