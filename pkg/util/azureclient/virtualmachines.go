package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"

	"github.com/openshift/openshift-azure/pkg/api"
)

func NewVirtualMachineScaleSetsClient(subscriptionID string, authorizer autorest.Authorizer, pluginConfig api.PluginConfig) VirtualMachineScaleSetsClient {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)
	return &azVirtualMachineScaleSetsClient{
		client: client,
	}
}

func NewVirtualMachineScaleSetVMsClient(subscriptionID string, authorizer autorest.Authorizer, pluginConfig api.PluginConfig) VirtualMachineScaleSetVMsClient {
	client := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)
	return &azVirtualMachineScaleSetVMsClient{
		client: client,
	}
}

func (az azVirtualMachineScaleSetVMsClient) GetClient() autorest.Client {
	return az.client.Client
}

func (az azVirtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) (result compute.VirtualMachineScaleSetVMListResultPage, err error) {
	return az.client.List(ctx, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand)
}

func (az azVirtualMachineScaleSetVMsClient) Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsDeleteFuture, err error) {
	return az.client.Delete(ctx, resourceGroupName, VMScaleSetName, instanceID)
}

func (az azVirtualMachineScaleSetsClient) GetClient() autorest.Client {
	return az.client.Client
}

func (az azVirtualMachineScaleSetsClient) Update(ctx context.Context, resourceGroupName string, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) (result compute.VirtualMachineScaleSetsUpdateFuture, err error) {
	return az.client.Update(ctx, resourceGroupName, VMScaleSetName, parameters)
}

func (az azVirtualMachineScaleSetVMsClient) Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsDeallocateFuture, err error) {
	return az.client.Deallocate(ctx, resourceGroupName, VMScaleSetName, instanceID)
}

func (az azVirtualMachineScaleSetsClient) UpdateInstances(ctx context.Context, resourceGroupName string, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) (result compute.VirtualMachineScaleSetsUpdateInstancesFuture, err error) {
	return az.client.UpdateInstances(ctx, resourceGroupName, VMScaleSetName, VMInstanceIDs)
}

func (az azVirtualMachineScaleSetVMsClient) Reimage(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsReimageFuture, err error) {
	return az.client.Reimage(ctx, resourceGroupName, VMScaleSetName, instanceID)
}

func (az azVirtualMachineScaleSetVMsClient) Start(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsStartFuture, err error) {
	return az.client.Start(ctx, resourceGroupName, VMScaleSetName, instanceID)
}
