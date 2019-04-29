package compute

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
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
