package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
)

func (cli *Client) ListScaleSets(ctx context.Context, resourceGroup string) ([]compute.VirtualMachineScaleSet, error) {
	pages, err := cli.VirtualMachineScaleSets.List(ctx, resourceGroup)
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

func (cli *Client) ListScaleSetVMs(ctx context.Context, resourceGroup, scaleSet string) ([]compute.VirtualMachineScaleSetVM, error) {
	pages, err := cli.VirtualMachineScaleSetVMs.List(ctx, resourceGroup, scaleSet, "", "", "")
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
