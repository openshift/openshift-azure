package cluster

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) *api.PluginError {
	// deployFn() may change the number of VMs.  If we can see that any VMs are
	// about to be deleted, drain them first.  Record which VMs are visible now
	// so that we can detect newly created VMs and wait for them to become ready.
	vmsBefore, err := u.getNodesAndDrain(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDrain}
	}
	err = deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	err = u.initialize(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitialize}
	}
	err = u.updateHash.GenerateNewHashes(azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepHashScaleSets}
	}
	err = managedcluster.WaitForHealthz(ctx, u.log, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	err = u.waitForNewNodes(ctx, cs, vmsBefore)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
	}
	if perr := u.updateInPlace(ctx, cs, api.AgentPoolProfileRoleMaster); perr != nil {
		return perr
	}
	if perr := u.updatePlusOne(ctx, cs, api.AgentPoolProfileRoleInfra); perr != nil {
		return perr
	}
	if perr := u.updatePlusOne(ctx, cs, api.AgentPoolProfileRoleCompute); perr != nil {
		return perr
	}
	return nil
}

func (u *simpleUpgrader) getNodesAndDrain(ctx context.Context, cs *api.OpenShiftManagedCluster) (map[kubeclient.ComputerName]struct{}, error) {
	vmsBefore := map[kubeclient.ComputerName]struct{}{}

	for _, agent := range cs.Properties.AgentPoolProfiles {
		vms, err := u.listVMs(ctx, cs, agent.Role)
		if err != nil {
			return nil, err
		}

		for i, vm := range vms {
			computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			if i < agent.Count {
				vmsBefore[computerName] = struct{}{}
			} else {
				err = u.delete(ctx, cs, agent.Role, *vm.InstanceID, computerName)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return vmsBefore, nil
}

func (u *simpleUpgrader) waitForNewNodes(ctx context.Context, cs *api.OpenShiftManagedCluster, nodes map[kubeclient.ComputerName]struct{}) (perr error) {
	err := u.updateHash.Reload()
	if err != nil {
		return err
	}
	defer func() {
		err := u.updateHash.Save()
		if err != nil {
			perr = err
		}
	}()

	existingVMs := make(map[updateblob.InstanceName]struct{})
	for _, agent := range cs.Properties.AgentPoolProfiles {
		vms, err := u.listVMs(ctx, cs, agent.Role)
		if err != nil {
			return err
		}

		// wait for newly created VMs to reach readiness
		for _, vm := range vms {
			computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			if _, found := nodes[computerName]; !found {
				u.log.Infof("waiting for %s to be ready", computerName)
				err = u.kubeclient.WaitForReady(ctx, agent.Role, computerName)
				if err != nil {
					return err
				}
				u.updateHash.UpdateInstanceHash(&vm)
			}
			// store all existing VMs in a map to compare against the VMs
			// stored in the blob in order to clean it up of stale VMs
			existingVMs[updateblob.InstanceName(*vm.Name)] = struct{}{}
		}
	}

	u.updateHash.DeleteAllBut(existingVMs)
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

func (u *simpleUpgrader) listVMs(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) ([]compute.VirtualMachineScaleSetVM, error) {
	vmPages, err := u.vmc.List(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), "", "", "")
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

// updatePlusOne creates new VMs and removes old VMs one by one.
func (u *simpleUpgrader) updatePlusOne(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (perr *api.PluginError) {
	count := getCount(cs, role)

	// store a list of all the VM instances now, so that if we end up creating
	// new ones (in the crash recovery case, we might not), we can detect which
	// they are
	oldVMs, err := u.listVMs(ctx, cs, role)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneListVMs}
	}

	// Filter out VMs that do not need to get upgraded. Should speed
	// up retrying failed upgrades.
	oldVMs, err = u.updateHash.FilterOldVMs(oldVMs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceReadBlob}
	}
	vmsBefore := map[string]struct{}{}
	for _, vm := range oldVMs {
		vmsBefore[*vm.InstanceID] = struct{}{}
	}

	defer func() {
		err := u.updateHash.Save()
		if err != nil {
			perr = &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceUpdateBlob}
		}
	}()

	for _, vm := range oldVMs {
		u.log.Infof("setting ss-%s capacity to %d", role, count+1)
		future, err := u.ssc.Update(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), compute.VirtualMachineScaleSetUpdate{
			Sku: &compute.Sku{
				Capacity: to.Int64Ptr(int64(count) + 1),
			},
		})
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneWaitForReady}
		}

		if err := future.WaitForCompletionRef(ctx, u.ssc.Client()); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneWaitForReady}
		}

		updatedList, err := u.listVMs(ctx, cs, role)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneListVMs}
		}

		// wait for newly created VMs to reach readiness (n.b. one alternative to
		// this approach would be for the CSE to not return until the node is
		// ready, but that is also problematic)
		for _, updated := range updatedList {
			if _, found := vmsBefore[*updated.InstanceID]; !found {
				computerName := kubeclient.ComputerName(*updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				u.log.Infof("waiting for %s to be ready", computerName)
				err = u.kubeclient.WaitForReady(ctx, role, computerName)
				if err != nil {
					return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneWaitForReady}
				}
				vmsBefore[*updated.InstanceID] = struct{}{}
				u.updateHash.UpdateInstanceHash(&updated)
			}
		}

		if err := u.delete(ctx, cs, role, *vm.InstanceID, kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneDeleteVMs}
		}

		u.updateHash.DeleteInstanceHash(updateblob.InstanceName(*vm.Name))
	}

	return nil
}

// updateInPlace updates one by one all the VMs of a scale set, in place.
func (u *simpleUpgrader) updateInPlace(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (perr *api.PluginError) {
	vms, err := u.listVMs(ctx, cs, role)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceListVMs}
	}

	sorted, err := u.sortMasterVMsByHealth(vms, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceSortMasters}
	}

	sorted, err = u.updateHash.FilterOldVMs(sorted)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceReadBlob}
	}

	defer func() {
		err := u.updateHash.Save()
		if err != nil {
			perr = &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceUpdateBlob}
		}
	}()

	for _, vm := range sorted {
		computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		u.log.Infof("draining %s", computerName)
		err = u.kubeclient.Drain(ctx, role, computerName)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceDrain}
		}

		{
			u.log.Infof("deallocating %s (%s)", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, *vm.InstanceID)
			future, err := u.vmc.Deallocate(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceDeallocate}
			}

			err = future.WaitForCompletionRef(ctx, u.vmc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceDeallocate}
			}
		}

		{
			u.log.Infof("updating %s", computerName)
			future, err := u.ssc.UpdateInstances(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
				InstanceIds: &[]string{*vm.InstanceID},
			})
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceUpdateVMs}
			}

			err = future.WaitForCompletionRef(ctx, u.ssc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceUpdateVMs}
			}
		}

		{
			u.log.Infof("reimaging %s", computerName)
			future, err := u.vmc.Reimage(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceReimage}
			}

			err = future.WaitForCompletionRef(ctx, u.vmc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceReimage}
			}
		}

		{
			u.log.Infof("starting %s", computerName)
			future, err := u.vmc.Start(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceStart}
			}

			err = future.WaitForCompletionRef(ctx, u.vmc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceStart}
			}
		}

		u.log.Infof("waiting for %s to be ready", computerName)
		err = u.kubeclient.WaitForReady(ctx, role, computerName)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceWaitForReady}
		}

		u.updateHash.UpdateInstanceHash(&vm)
	}

	return nil
}

func (u *simpleUpgrader) sortMasterVMsByHealth(vms []compute.VirtualMachineScaleSetVM, cs *api.OpenShiftManagedCluster) ([]compute.VirtualMachineScaleSetVM, error) {
	var ready, unready []compute.VirtualMachineScaleSetVM
	for _, vm := range vms {
		nodeName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		isReady, err := u.kubeclient.MasterIsReady(nodeName)
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

func (u *simpleUpgrader) delete(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, instanceID string, nodeName kubeclient.ComputerName) error {
	u.log.Infof("draining %s", nodeName)
	if err := u.kubeclient.Drain(ctx, role, nodeName); err != nil {
		return err
	}

	u.log.Infof("deleting %s", nodeName)
	future, err := u.vmc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, u.vmc.Client())
}
