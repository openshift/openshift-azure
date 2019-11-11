package network

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"

	"github.com/openshift/openshift-azure/pkg/arm/constants"
)

// FakeVirtualNetworksClient is a Fake of NetworkClient interface
type FakeVirtualNetworksClient struct {
}

// NewFakeVirtualNetworksClient creates a new Fake instance
func NewFakeVirtualNetworksClient() *FakeVirtualNetworksClient {
	return &FakeVirtualNetworksClient{}
}

func (vn *FakeVirtualNetworksClient) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, expand string) (network.VirtualNetwork, error) {
	return network.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			DhcpOptions: &network.DhcpOptions{
				DNSServers: &[]string{constants.AzureNameserver},
			},
		},
	}, nil
}

func (vn *FakeVirtualNetworksClient) List(ctx context.Context, resourceGroupName string) (network.VirtualNetworkListResultPage, error) {
	return network.VirtualNetworkListResultPage{}, nil
}

func (vn *FakeVirtualNetworksClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters network.VirtualNetwork) (network.VirtualNetworksCreateOrUpdateFuture, error) {
	return network.VirtualNetworksCreateOrUpdateFuture{}, nil
}

func (vn *FakeVirtualNetworksClient) Delete(ctx context.Context, resourceGroupName string, virtualNetworkName string) (network.VirtualNetworksDeleteFuture, error) {
	return network.VirtualNetworksDeleteFuture{}, nil
}

type fakeClient struct {
}

func (f *fakeClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200}, nil
}

func (d *FakeVirtualNetworksClient) Client() autorest.Client {
	return autorest.Client{Sender: &fakeClient{}}
}
