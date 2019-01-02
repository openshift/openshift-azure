package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// VirtualMachineScaleSetsClient is a minimal interface for azure VirtualMachineScaleSetsClient
type VirtualMachineScaleSetsClient interface {
	VirtualMachineScaleSetsClientAddons
	CreateOrUpdate(ctx context.Context, resourceGroupName string, VMScaleSetName string, parameters compute.VirtualMachineScaleSet) (compute.VirtualMachineScaleSetsCreateOrUpdateFuture, error)
	Update(ctx context.Context, resourceGroupName string, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) (compute.VirtualMachineScaleSetsUpdateFuture, error)
	UpdateInstances(ctx context.Context, resourceGroupName string, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) (compute.VirtualMachineScaleSetsUpdateInstancesFuture, error)
	Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string) (compute.VirtualMachineScaleSetsDeleteFuture, error)
	Client
}

type virtualMachineScaleSetsClient struct {
	compute.VirtualMachineScaleSetsClient
}

var _ VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient{}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient
func NewVirtualMachineScaleSetsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) VirtualMachineScaleSetsClient {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	setupClient(&client.Client, authorizer, languages)

	return &virtualMachineScaleSetsClient{
		VirtualMachineScaleSetsClient: client,
	}
}

func (c *virtualMachineScaleSetsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetsClient.Client
}

// VirtualMachineScaleSetVMsClient is a minimal interface for azure VirtualMachineScaleSetVMsClient
type VirtualMachineScaleSetVMsClient interface {
	VirtualMachineScaleSetVMsClientAddons
	Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsDeleteFuture, error)
	Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsDeallocateFuture, error)
	Reimage(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsReimageFuture, error)
	Restart(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsRestartFuture, err error)
	Start(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsStartFuture, error)
	Client
}

type virtualMachineScaleSetVMsClient struct {
	compute.VirtualMachineScaleSetVMsClient
}

var _ VirtualMachineScaleSetVMsClient = &virtualMachineScaleSetVMsClient{}

// NewVirtualMachineScaleSetVMsClient creates a new VirtualMachineScaleSetVMsClient
func NewVirtualMachineScaleSetVMsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) VirtualMachineScaleSetVMsClient {
	client := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
	setupClient(&client.Client, authorizer, languages)

	return &virtualMachineScaleSetVMsClient{
		VirtualMachineScaleSetVMsClient: client,
	}
}

func (c *virtualMachineScaleSetVMsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetVMsClient.Client
}

// VirtualMachineScaleSetExtensionsClient is a minimal interface for azure VirtualMachineScaleSetExtensionsClient
type VirtualMachineScaleSetExtensionsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, VMScaleSetName string, vmssExtensionName string, extensionParameters compute.VirtualMachineScaleSetExtension) (result compute.VirtualMachineScaleSetExtensionsCreateOrUpdateFuture, err error)
	Get(ctx context.Context, resourceGroupName string, VMScaleSetName string, vmssExtensionName string, expand string) (result compute.VirtualMachineScaleSetExtension, err error)
	List(ctx context.Context, resourceGroupName string, VMScaleSetName string) (result compute.VirtualMachineScaleSetExtensionListResultPage, err error)
	Client
}

type virtualMachineScaleSetExtensionsClient struct {
	compute.VirtualMachineScaleSetExtensionsClient
}

var _ VirtualMachineScaleSetExtensionsClient = &virtualMachineScaleSetExtensionsClient{}

// NewVirtualMachineScaleSetExtensionsClient creates a new VirtualMachineScaleSetExtensionsClient
func NewVirtualMachineScaleSetExtensionsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) VirtualMachineScaleSetExtensionsClient {
	client := compute.NewVirtualMachineScaleSetExtensionsClient(subscriptionID)
	setupClient(&client.Client, authorizer, languages)

	return &virtualMachineScaleSetExtensionsClient{
		VirtualMachineScaleSetExtensionsClient: client,
	}
}

func (c *virtualMachineScaleSetExtensionsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetExtensionsClient.Client
}
