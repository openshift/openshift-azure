package network

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// InterfacesClient is a minimal interface for azure NewInterfacesClient
type InterfacesClient interface {
	Get(ctx context.Context, resourceGroupName string, networkInterfaceName string, expand string) (network.Interface, error)
	GetVirtualMachineScaleSetNetworkInterface(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, virtualmachineIndex string, networkInterfaceName string, expand string) (result network.Interface, err error)
}

type interfacesClient struct {
	network.InterfacesClient
}

var _ InterfacesClient = &interfacesClient{}

// NewInterfacesClient creates a new InterfacesClient
func NewInterfacesClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) InterfacesClient {
	client := network.NewInterfacesClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "network.InterfacesClient", &client.Client, authorizer)

	return &interfacesClient{
		InterfacesClient: client,
	}
}
