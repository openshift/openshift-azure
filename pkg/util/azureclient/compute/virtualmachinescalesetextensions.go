package compute

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// VirtualMachineScaleSetExtensionsClient is a minimal interface for azure VirtualMachineScaleSetExtensionsClient
type VirtualMachineScaleSetExtensionsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, VMScaleSetName string, vmssExtensionName string, extensionParameters compute.VirtualMachineScaleSetExtension) (result compute.VirtualMachineScaleSetExtensionsCreateOrUpdateFuture, err error)
	Get(ctx context.Context, resourceGroupName string, VMScaleSetName string, vmssExtensionName string, expand string) (result compute.VirtualMachineScaleSetExtension, err error)
	List(ctx context.Context, resourceGroupName string, VMScaleSetName string) (result compute.VirtualMachineScaleSetExtensionListResultPage, err error)
	azureclient.Client
}

type virtualMachineScaleSetExtensionsClient struct {
	compute.VirtualMachineScaleSetExtensionsClient
}

var _ VirtualMachineScaleSetExtensionsClient = &virtualMachineScaleSetExtensionsClient{}

// NewVirtualMachineScaleSetExtensionsClient creates a new VirtualMachineScaleSetExtensionsClient
func NewVirtualMachineScaleSetExtensionsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetExtensionsClient {
	client := compute.NewVirtualMachineScaleSetExtensionsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "compute.VirtualMachineScaleSetExtensionsClient", &client.Client, authorizer)

	return &virtualMachineScaleSetExtensionsClient{
		VirtualMachineScaleSetExtensionsClient: client,
	}
}

func (c *virtualMachineScaleSetExtensionsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetExtensionsClient.Client
}
