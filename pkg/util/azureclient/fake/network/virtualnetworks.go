package network

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"
)

type NetworkRP struct {
	Nameservers *[]string
}

// FakeVirtualNetworksClient is a Fake of NetworkClient interface
type FakeVirtualNetworksClient struct {
	NetworkRP
}

// NewFakeVirtualNetworksClient creates a new Fake instance
func NewFakeVirtualNetworksClient(rp *NetworkRP) *FakeVirtualNetworksClient {
	return &FakeVirtualNetworksClient{NetworkRP: *rp}
}

func (vn *FakeVirtualNetworksClient) Get(ctx context.Context, resourceGroupName string, virtualNetworkName string, expand string) (network.VirtualNetwork, error) {
	return network.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			DhcpOptions: &network.DhcpOptions{
				DNSServers: vn.Nameservers,
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
