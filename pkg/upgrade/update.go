package upgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Update(ctx context.Context, cs, oldCs *api.OpenShiftManagedCluster, azuredeploy []byte) error {
	config := auth.NewClientCredentialsConfig(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
	authorizer, err := config.Authorizer()
	if err != nil {
		return err
	}
	ssc := compute.NewVirtualMachineScaleSetsClient(cs.Properties.AzProfile.SubscriptionID)
	ssc.Authorizer = authorizer
	vmc := compute.NewVirtualMachineScaleSetVMsClient(cs.Properties.AzProfile.SubscriptionID)
	vmc.Authorizer = authorizer

	// Need to determine whether the current update is a scale operation before
	// applying the ARM template. For scale up, we need to figure out which are
	// the new VMs that were created and wait for them to turn ready. For scale
	// down, we need to drain the correct VMs before applying the ARM template
	// scales them down.
	isScaleOp := isScaleUpdate(cs, oldCs)
	vmsBefore := map[string]struct{}{}
	if isScaleOp {
		for _, agent := range cs.Properties.AgentPoolProfiles {
			vms, err := listVMs(ctx, cs, vmc, agent.Role)
			if err != nil {
				return err
			}

			if len(vms) > agent.Count {
				for _, vm := range vms[agent.Count:] {
					if err := u.delete(ctx, cs, vmc, agent.Role, *vm.InstanceID, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName); err != nil {
						return err
					}
				}
			} else {
				for _, vm := range vms {
					vmsBefore[*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName] = struct{}{}
				}
			}
		}
	}

	// Apply the ARM template
	if err := Deploy(ctx, cs, u.Initializer, azuredeploy); err != nil {
		return err
	}

	if isScaleOp {
		for _, agent := range cs.Properties.AgentPoolProfiles {
			vms, err := listVMs(ctx, cs, vmc, agent.Role)
			if err != nil {
				return err
			}

			// wait for newly created VMs to reach readiness
			for _, vm := range vms {
				hostname := *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
				if _, found := vmsBefore[hostname]; !found {
					log.Infof("waiting for %s to be ready", hostname)
					err = u.WaitForReady(ctx, cs, agent.Role, hostname)
					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	err = u.updateInPlace(ctx, cs, ssc, vmc, api.AgentPoolProfileRoleMaster)
	if err != nil {
		return err
	}

	// TODO: updatePlusOne isn't good enough to avoid interruption on our infra
	// nodes.
	err = u.updatePlusOne(ctx, cs, ssc, vmc, api.AgentPoolProfileRoleInfra)
	if err != nil {
		return err
	}

	err = u.updatePlusOne(ctx, cs, ssc, vmc, api.AgentPoolProfileRoleCompute)
	if err != nil {
		return err
	}

	return nil
}

// isScaleUpdate returns whether the update is a scale operation.
func isScaleUpdate(cs, oldCs *api.OpenShiftManagedCluster) bool {
	newCounts := make(map[string]int)
	for _, new := range cs.Properties.AgentPoolProfiles {
		newCounts[new.Name] = new.Count
	}

	var isScaleUpdate bool
	for i, old := range oldCs.Properties.AgentPoolProfiles {
		if newCounts[old.Name] != old.Count {
			isScaleUpdate = true
		}
		oldCs.Properties.AgentPoolProfiles[i].Count = newCounts[old.Name]
	}

	// No scale operations.
	if !isScaleUpdate {
		return false
	}
	// Includes other non-scaling related updates
	if !reflect.DeepEqual(cs, oldCs) {
		return false
	}

	return true
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
func (u *simpleUpgrader) updatePlusOne(ctx context.Context, cs *api.OpenShiftManagedCluster, ssc compute.VirtualMachineScaleSetsClient, vmc compute.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) error {
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
			if _, found := vmsBefore[*updated.InstanceID]; !found {
				log.Infof("waiting for %s to be ready", *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				err = u.WaitForReady(ctx, cs, role, *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
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
	vms, err := listVMs(ctx, cs, vmc, role)
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
		err = u.WaitForReady(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
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

	return future.WaitForCompletion(ctx, vmc.Client)
}
