package main

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

func update(ctx context.Context, cs *api.OpenShiftManagedCluster, p api.Plugin) error {
	config := auth.NewClientCredentialsConfig(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
	authorizer, err := config.Authorizer()
	if err != nil {
		return err
	}

	ssc := compute.NewVirtualMachineScaleSetsClient(cs.Properties.AzProfile.SubscriptionID)
	ssc.Authorizer = authorizer
	vmc := compute.NewVirtualMachineScaleSetVMsClient(cs.Properties.AzProfile.SubscriptionID)
	vmc.Authorizer = authorizer

	err = updateInPlace(ctx, cs, p, ssc, vmc, api.AgentPoolProfileRoleMaster)
	if err != nil {
		return err
	}

	// TODO: updatePlusOne isn't good enough to avoid interruption on our infra
	// nodes.
	err = updatePlusOne(ctx, cs, p, ssc, vmc, api.AgentPoolProfileRoleInfra)
	if err != nil {
		return err
	}

	err = updatePlusOne(ctx, cs, p, ssc, vmc, api.AgentPoolProfileRoleCompute)
	if err != nil {
		return err
	}

	return nil
}

func getCount(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) int {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			return app.Count
		}
	}

	panic("invalid role")
}

func listVMs(ctx context.Context, cs *api.OpenShiftManagedCluster, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) ([]compute.VirtualMachineScaleSetVM, error) {
	vmPages, err := vmc.List(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), "", "", "")
	if err != nil {
		return nil, err
	}

	var vms []compute.VirtualMachineScaleSetVM
	for vmPages.NotDone() {
		vms = append(vms, vmPages.Values()...)

		err = vmPages.Next()
		if err != nil {
			return nil, err
		}
	}

	return vms, nil
}

// updatePlusOne creates an extra VM, then runs updateInPlace, then removes the
// extra VM.
func updatePlusOne(ctx context.Context, cs *api.OpenShiftManagedCluster, p api.Plugin, ssc compute.VirtualMachineScaleSetsClient, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) error {
	count := getCount(cs, role)

	// store a list of all the VM instances now, so that if we end up creating
	// new ones (in the crash recovery case, we might not), we can detect which
	// they are
	oldVMs, err := listVMs(ctx, cs, vmc, role)
	if err != nil {
		return err
	}

	// TODO: Filter out VMs that do not need to get upgraded. Should speed
	// up retrying failed upgrades.
	vmsBefore := map[string]struct{}{}
	for _, vm := range oldVMs {
		vmsBefore[*vm.InstanceID] = struct{}{}
	}
	vmsUpgraded := map[string]struct{}{}

	for _, vm := range oldVMs {
		log.Infof("setting ss-%s capacity to %d", role, count+1)
		future, err := ssc.Update(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), compute.VirtualMachineScaleSetUpdate{
			Sku: &compute.Sku{
				Capacity: to.Int64Ptr(int64(count) + 1),
			},
		})
		if err != nil {
			return err
		}

		if err := future.WaitForCompletion(ctx, ssc.Client); err != nil {
			return err
		}

		updatedList, err := listVMs(ctx, cs, vmc, role)
		if err != nil {
			return err
		}

		// wait for newly created VMs to reach readiness (n.b. one alternative to
		// this approach would be for the CSE to not return until the node is
		// ready, but that is also problematic)
		for _, updated := range updatedList {
			_, updatedFound := vmsUpgraded[*updated.InstanceID]
			if _, oldFound := vmsBefore[*updated.InstanceID]; !oldFound && !updatedFound {
				log.Infof("waiting for %s to be ready", *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				err = p.WaitForReady(ctx, cs, role, *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				if err != nil {
					return err
				}
				vmsUpgraded[*updated.InstanceID] = struct{}{}
			}
		}

		// remove surplus VM
		log.Infof("draining %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		err = p.Drain(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		if err != nil {
			return err
		}

		log.Infof("deleting %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		delFuture, err := vmc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
		if err != nil {
			return err
		}

		if err := delFuture.WaitForCompletion(ctx, vmc.Client); err != nil {
			return err
		}
	}

	return nil
}

// updateInPlace updates one by one all the VMs of a scale set, in place.
func updateInPlace(ctx context.Context, cs *api.OpenShiftManagedCluster, p api.Plugin, ssc compute.VirtualMachineScaleSetsClient, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) error {
	vms, err := listVMs(ctx, cs, vmc, role)
	if err != nil {
		return err
	}

	for _, vm := range vms {
		log.Infof("draining %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		err = p.Drain(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		if err != nil {
			return err
		}

		{
			log.Infof("deallocating %s (%s)", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, *vm.InstanceID)
			future, err := vmc.Deallocate(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return err
			}

			err = future.WaitForCompletion(ctx, vmc.Client)
			if err != nil {
				return err
			}
		}

		{
			log.Infof("updating %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			future, err := ssc.UpdateInstances(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
				InstanceIds: &[]string{*vm.InstanceID},
			})
			if err != nil {
				return err
			}

			err = future.WaitForCompletion(ctx, ssc.Client)
			if err != nil {
				return err
			}
		}

		{
			log.Infof("reimaging %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			future, err := vmc.Reimage(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return err
			}

			err = future.WaitForCompletion(ctx, vmc.Client)
			if err != nil {
				return err
			}
		}

		{
			log.Infof("starting %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			future, err := vmc.Start(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return err
			}

			err = future.WaitForCompletion(ctx, vmc.Client)
			if err != nil {
				return err
			}
		}

		log.Infof("waiting for %s to be ready", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		err = p.WaitForReady(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		if err != nil {
			return err
		}
	}

	return nil
}
