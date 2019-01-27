package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
)

// NOTE: we should be very sparing and have a high bar for these kind of hacks.

type VirtualMachineScaleSetsClientAddons interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSet) error
	Delete(ctx context.Context, resourceGroupName, VMScaleSetName string) error
	List(ctx context.Context, resourceGroup string) ([]compute.VirtualMachineScaleSet, error)
	Update(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) error
	UpdateInstances(ctx context.Context, resourceGroupName, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) error
}

func (c *virtualMachineScaleSetsClient) CreateOrUpdate(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSet) error {
	future, err := c.VirtualMachineScaleSetsClient.CreateOrUpdate(ctx, resourceGroupName, VMScaleSetName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetsClient.Client)
}

func (c *virtualMachineScaleSetsClient) Delete(ctx context.Context, resourceGroupName, VMScaleSetName string) error {
	future, err := c.VirtualMachineScaleSetsClient.Delete(ctx, resourceGroupName, VMScaleSetName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetsClient.Client)
}

func (c *virtualMachineScaleSetsClient) List(ctx context.Context, resourceGroupName string) ([]compute.VirtualMachineScaleSet, error) {
	pages, err := c.VirtualMachineScaleSetsClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	var scaleSets []compute.VirtualMachineScaleSet
	for pages.NotDone() {
		scaleSets = append(scaleSets, pages.Values()...)

		err = pages.Next()
		if err != nil {
			return nil, err
		}
	}

	return scaleSets, nil
}

func (c *virtualMachineScaleSetsClient) Update(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) error {
	future, err := c.VirtualMachineScaleSetsClient.Update(ctx, resourceGroupName, VMScaleSetName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetsClient.Client)
}

func (c *virtualMachineScaleSetsClient) UpdateInstances(ctx context.Context, resourceGroupName, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) error {
	future, err := c.VirtualMachineScaleSetsClient.UpdateInstances(ctx, resourceGroupName, VMScaleSetName, VMInstanceIDs)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetsClient.Client)
}

type VirtualMachineScaleSetVMsClientAddons interface {
	Deallocate(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
	Delete(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
	List(ctx context.Context, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand string) ([]compute.VirtualMachineScaleSetVM, error)
	Reimage(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
	Restart(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error
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

func (c *virtualMachineScaleSetVMsClient) Reimage(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	future, err := c.VirtualMachineScaleSetVMsClient.Reimage(ctx, resourceGroupName, VMScaleSetName, instanceID, nil)
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

func (c *virtualMachineScaleSetVMsClient) Start(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	future, err := c.VirtualMachineScaleSetVMsClient.Start(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.VirtualMachineScaleSetVMsClient.Client)
}
