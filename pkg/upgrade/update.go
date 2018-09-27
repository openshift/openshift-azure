package upgrade

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte) error {
	clients, err := u.getClients(ctx, cs)
	if err != nil {
		return err
	}

	// Deploy() may change the number of VMs.  If we can see that any VMs are
	// about to be deleted, drain them first.  Record which VMs are visible now
	// so that we can detect newly created VMs and wait for them to become
	// ready.

	vmsBefore := map[string]struct{}{}

	for _, agent := range cs.Properties.AgentPoolProfiles {
		vms, err := ListVMs(ctx, cs, clients.VirtualMachineScaleSetVMs, agent.Role)
		if err != nil {
			return err
		}

		for i, vm := range vms {
			if i < agent.Count {
				vmsBefore[*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName] = struct{}{}

			} else {
				err = u.delete(ctx, cs, clients.VirtualMachineScaleSetVMs, agent.Role, *vm.InstanceID, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				if err != nil {
					return err
				}
			}
		}
	}

	err = u.pluginConfig.Deployer(ctx, cs, azuredeploy, u.pluginConfig)
	if err != nil {
		return err
	}

	err = u.InitializeCluster(ctx, cs)
	if err != nil {
		return err
	}

	err = u.postDeployWaitForAll(ctx, cs, vmsBefore)
	if err != nil {
		return err
	}

	// For PP day 1, scale is permitted but not any other sort of update.  When
	// we enable configuration changes and/or upgrades, uncomment this code.  At
	// the same time, current thinking is that we will add a hash-based
	// mechanism to avoid unnecessary VM rotations as well.

	if os.Getenv("RUNNING_UNDER_TEST") != "" {
		err = u.updateInPlace(ctx, cs, clients.VirtualMachineScaleSets, clients.VirtualMachineScaleSetVMs, api.AgentPoolProfileRoleMaster)
		if err != nil {
			return err
		}

		err = u.updatePlusOne(ctx, cs, clients.VirtualMachineScaleSets, clients.VirtualMachineScaleSetVMs, api.AgentPoolProfileRoleInfra)
		if err != nil {
			return err
		}

		err = u.updatePlusOne(ctx, cs, clients.VirtualMachineScaleSets, clients.VirtualMachineScaleSetVMs, api.AgentPoolProfileRoleCompute)
		if err != nil {
			return err
		}
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

func ListVMs(ctx context.Context, cs *api.OpenShiftManagedCluster, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) ([]compute.VirtualMachineScaleSetVM, error) {
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
func (u *simpleUpgrader) updatePlusOne(ctx context.Context, cs *api.OpenShiftManagedCluster, ssc compute.VirtualMachineScaleSetsClient, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) error {
	count := getCount(cs, role)

	// store a list of all the VM instances now, so that if we end up creating
	// new ones (in the crash recovery case, we might not), we can detect which
	// they are
	oldVMs, err := ListVMs(ctx, cs, vmc, role)
	if err != nil {
		return err
	}

	// TODO: Filter out VMs that do not need to get upgraded. Should speed
	// up retrying failed upgrades.
	vmsBefore := map[string]struct{}{}
	for _, vm := range oldVMs {
		vmsBefore[*vm.InstanceID] = struct{}{}
	}

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

		if err := future.WaitForCompletionRef(ctx, ssc.Client); err != nil {
			return err
		}

		updatedList, err := ListVMs(ctx, cs, vmc, role)
		if err != nil {
			return err
		}

		// wait for newly created VMs to reach readiness (n.b. one alternative to
		// this approach would be for the CSE to not return until the node is
		// ready, but that is also problematic)
		for _, updated := range updatedList {
			if _, found := vmsBefore[*updated.InstanceID]; !found {
				log.Infof("waiting for %s to be ready", *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				err = WaitForReady(ctx, cs, role, *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				if err != nil {
					return err
				}
				vmsBefore[*updated.InstanceID] = struct{}{}
			}
		}

		if err := u.delete(ctx, cs, vmc, role, *vm.InstanceID, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName); err != nil {
			return err
		}
	}

	return nil
}

// updateInPlace updates one by one all the VMs of a scale set, in place.
func (u *simpleUpgrader) updateInPlace(ctx context.Context, cs *api.OpenShiftManagedCluster, ssc compute.VirtualMachineScaleSetsClient, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) error {
	vms, err := ListVMs(ctx, cs, vmc, role)
	if err != nil {
		return err
	}

	sorted, err := sortMasterVMsByHealth(vms, cs)
	if err != nil {
		return err
	}

	for _, vm := range sorted {
		log.Infof("draining %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		err = u.drain(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		if err != nil {
			return err
		}

		{
			log.Infof("deallocating %s (%s)", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, *vm.InstanceID)
			future, err := vmc.Deallocate(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return err
			}

			err = future.WaitForCompletionRef(ctx, vmc.Client)
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

			err = future.WaitForCompletionRef(ctx, ssc.Client)
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

			err = future.WaitForCompletionRef(ctx, vmc.Client)
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

			err = future.WaitForCompletionRef(ctx, vmc.Client)
			if err != nil {
				return err
			}
		}

		log.Infof("waiting for %s to be ready", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		err = WaitForReady(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		if err != nil {
			return err
		}
	}

	return nil
}

func sortMasterVMsByHealth(vms []compute.VirtualMachineScaleSetVM, cs *api.OpenShiftManagedCluster) ([]compute.VirtualMachineScaleSetVM, error) {
	kc, err := managedcluster.ClientsetFromConfig(cs)
	if err != nil {
		return nil, err
	}

	var ready, unready []compute.VirtualMachineScaleSetVM
	for _, vm := range vms {
		nodeName := *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
		isReady, err := masterIsReady(kc, nodeName)
		if err != nil {
			return nil, fmt.Errorf("cannot get health for %q: %v", nodeName, err)
		}
		if isReady {
			ready = append(ready, vm)
		} else {
			unready = append(unready, vm)
		}
	}

	return append(unready, ready...), nil
}

func (u *simpleUpgrader) delete(ctx context.Context, cs *api.OpenShiftManagedCluster, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole, instanceID, nodeName string) error {
	log.Infof("draining %s", nodeName)
	if err := u.drain(ctx, cs, role, nodeName); err != nil {
		return err
	}

	log.Infof("deleting %s", nodeName)
	future, err := vmc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, vmc.Client)
}
