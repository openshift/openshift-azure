package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

// VirtualMachineScaleSetsClient is a minimal interface for azure VirtualMachineScaleSetsClient
type VirtualMachineScaleSetsClient interface {
	Update(ctx context.Context, resourceGroupName string, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) (compute.VirtualMachineScaleSetsUpdateFuture, error)
	UpdateInstances(ctx context.Context, resourceGroupName string, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) (compute.VirtualMachineScaleSetsUpdateInstancesFuture, error)
	Client
}

type virtualMachineScaleSetsClient struct {
	compute.VirtualMachineScaleSetsClient
}

var _ VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient{}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient
func NewVirtualMachineScaleSetsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) VirtualMachineScaleSetsClient {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)

	return &virtualMachineScaleSetsClient{
		VirtualMachineScaleSetsClient: client,
	}
}

func (c *virtualMachineScaleSetsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetsClient.Client
}

// VirtualMachineScaleSetVMsClient is a minimal interface for azure VirtualMachineScaleSetVMsClient
type VirtualMachineScaleSetVMsClient interface {
	Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsDeleteFuture, error)
	Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsDeallocateFuture, error)
	List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) (compute.VirtualMachineScaleSetVMListResultPage, error)
	Reimage(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsReimageFuture, error)
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
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)

	return &virtualMachineScaleSetVMsClient{
		VirtualMachineScaleSetVMsClient: client,
	}
}

func (c *virtualMachineScaleSetVMsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetVMsClient.Client
}
