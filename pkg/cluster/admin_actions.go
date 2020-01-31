package cluster

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
)

func (u *Upgrade) Restart(ctx context.Context, scaleset, instanceID string) error {
	return u.Vmc.Restart(ctx, u.Cs.Properties.AzProfile.ResourceGroup, scaleset, instanceID)
}

func (u *Upgrade) Reimage(ctx context.Context, scaleset, instanceID string) error {
	return u.Vmc.Reimage(ctx, u.Cs.Properties.AzProfile.ResourceGroup, scaleset, instanceID, nil)
}

func (u *Upgrade) ListVMHostnames(ctx context.Context) ([]string, error) {
	scalesets, err := u.Ssc.List(ctx, u.Cs.Properties.AzProfile.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var hostnames []string
	for _, ss := range scalesets {
		vms, err := u.Vmc.List(ctx, u.Cs.Properties.AzProfile.ResourceGroup, *ss.Name, "", "", "")
		if err != nil {
			return nil, err
		}

		for _, vm := range vms {
			hostnames = append(hostnames, strings.ToLower(*vm.OsProfile.ComputerName))
		}
	}

	return hostnames, nil
}

func (u *Upgrade) GetImageVerInfo(ctx context.Context) ([]string, error) {
	scalesets, err := u.Ssc.List(ctx, u.Cs.Properties.AzProfile.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var imagever []string
	for _, ss := range scalesets {
		vms, err := u.Vmc.List(ctx, u.Cs.Properties.AzProfile.ResourceGroup, *ss.Name, "", "", "")
		if err != nil {
			return nil, err
		}

		for _, vm := range vms {
			imagever = append(imagever, strings.ToLower(*vm.VirtualMachineScaleSetVMProperties.StorageProfile.ImageReference.Version))
		}
	}

	return imagever, nil
}

func (u *Upgrade) RunCommand(ctx context.Context, scaleset, instanceID, command string) error {
	return u.Vmc.RunCommand(ctx, u.Cs.Properties.AzProfile.ResourceGroup, scaleset, instanceID, compute.RunCommandInput{
		CommandID: to.StringPtr("RunShellScript"),
		Script:    &[]string{command},
	})
}
