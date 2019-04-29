package compute

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
)

// NOTE: we should be very sparing and have a high bar for these kind of hacks.

type VirtualMachineScaleSetVMsClientAddons interface {
	Deallocate(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
	Delete(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
	List(ctx context.Context, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand string) ([]compute.VirtualMachineScaleSetVM, error)
	Reimage(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string, VMScaleSetVMReimageInput *compute.VirtualMachineScaleSetVMReimageParameters) error
	Restart(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
	RunCommand(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters compute.RunCommandInput) error
	Start(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
}

func (c *virtualMachineScaleSetVMsClient) Deallocate(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	future, err := c.VirtualMachineScaleSetVMsClient.Deallocate(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}

func (c *virtualMachineScaleSetVMsClient) Delete(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	future, err := c.VirtualMachineScaleSetVMsClient.Delete(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}

func (c *virtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand string) ([]compute.VirtualMachineScaleSetVM, error) {
	pages, err := c.VirtualMachineScaleSetVMsClient.List(ctx, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand)
	if err != nil {
		return nil, err
	}

	var vms []compute.VirtualMachineScaleSetVM
	for pages.NotDone() {
		vms = append(vms, pages.Values()...)

		err = pages.Next()
		if err != nil {
			return nil, err
		}
	}

	return vms, nil
}

func (c *virtualMachineScaleSetVMsClient) Reimage(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string, VMScaleSetVMReimageInput *compute.VirtualMachineScaleSetVMReimageParameters) error {
	future, err := c.VirtualMachineScaleSetVMsClient.Reimage(ctx, resourceGroupName, VMScaleSetName, instanceID, VMScaleSetVMReimageInput)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}

func (c *virtualMachineScaleSetVMsClient) Restart(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	future, err := c.VirtualMachineScaleSetVMsClient.Restart(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}

func (c *virtualMachineScaleSetVMsClient) RunCommand(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters compute.RunCommandInput) error {
	future, err := c.VirtualMachineScaleSetVMsClient.RunCommand(ctx, resourceGroupName, VMScaleSetName, instanceID, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}

func (c *virtualMachineScaleSetVMsClient) Start(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	future, err := c.VirtualMachineScaleSetVMsClient.Start(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}
