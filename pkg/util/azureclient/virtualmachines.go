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

func (az azVirtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) (result compute.VirtualMachineScaleSetVMListResultPage, err error) {
	return az.client.List(ctx, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand)
}

func (az azVirtualMachineScaleSetVMsClient) Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsDeleteFuture, err error) {
	future, err := az.client.Delete(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return result, err
	}
	err = future.WaitForCompletionRef(ctx, az.client.Client)
	if err != nil {
		return future, err
	}
	return future, err
}

func (az azVirtualMachineScaleSetsClient) Update(ctx context.Context, resourceGroupName string, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) (result compute.VirtualMachineScaleSetsUpdateFuture, err error) {
	future, err := az.client.Update(ctx, resourceGroupName, VMScaleSetName, parameters)
	if err != nil {
		return result, err
	}
	err = future.WaitForCompletionRef(ctx, az.client.Client)
	if err != nil {
		return future, err
	}
	return future, err
}

func (az azVirtualMachineScaleSetVMsClient) Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsDeallocateFuture, err error) {
	future, err := az.client.Deallocate(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return result, err
	}

	err = future.WaitForCompletionRef(ctx, az.client.Client)
	if err != nil {
		return result, err
	}
	return future, err
}

func (az azVirtualMachineScaleSetsClient) UpdateInstances(ctx context.Context, resourceGroupName string, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) (result compute.VirtualMachineScaleSetsUpdateInstancesFuture, err error) {
	future, err := az.client.UpdateInstances(ctx, resourceGroupName, VMScaleSetName, VMInstanceIDs)
	if err != nil {
		return result, err
	}

	err = future.WaitForCompletionRef(ctx, az.client.Client)
	if err != nil {
		return result, err
	}
	return future, err
}

func (az azVirtualMachineScaleSetVMsClient) Reimage(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsReimageFuture, err error) {
	future, err := az.client.Reimage(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return result, err
	}

	err = future.WaitForCompletionRef(ctx, az.client.Client)
	if err != nil {
		return result, err
	}
	return future, err
}

func (az azVirtualMachineScaleSetVMsClient) Start(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (result compute.VirtualMachineScaleSetVMsStartFuture, err error) {
	future, err := az.client.Start(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return result, err
	}

	err = future.WaitForCompletionRef(ctx, az.client.Client)
	if err != nil {
		return result, err
	}
	return future, err
}
