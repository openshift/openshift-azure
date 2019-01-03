package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
)

// NOTE: we should be very sparing and have a high bar for these kind of hacks.

type VirtualMachineScaleSetsClientAddons interface {
	List(ctx context.Context, resourceGroup string) ([]compute.VirtualMachineScaleSet, error)
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

type VirtualMachineScaleSetVMsClientAddons interface {
	List(ctx context.Context, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand string) ([]compute.VirtualMachineScaleSetVM, error)
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
