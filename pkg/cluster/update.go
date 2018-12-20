package cluster

import (
	"context"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/config"
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
	ssHashes, err := u.hashScaleSets(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepHashScaleSets}
	}
	err = managedcluster.WaitForHealthz(ctx, u.log, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	err = u.waitForNewNodes(ctx, cs, vmsBefore, ssHashes)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
	}
	if perr := u.updateMasterAgentPool(ctx, cs, ssHashes); perr != nil {
		return perr
	}
	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra) {
		if perr := u.updatePlusOne(ctx, cs, &app, ssHashes); perr != nil {
			return perr
		}
	}
	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute) {
		if perr := u.updatePlusOne(ctx, cs, &app, ssHashes); perr != nil {
			return perr
		}
	}
	return nil
}

func (u *simpleUpgrader) getNodesAndDrain(ctx context.Context, cs *api.OpenShiftManagedCluster) (map[kubeclient.ComputerName]struct{}, error) {
	vmsBefore := map[kubeclient.ComputerName]struct{}{}

	for _, app := range cs.Properties.AgentPoolProfiles {
		vms, err := u.listVMs(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(app.Name))
		if err != nil {
			return nil, err
		}

		for i, vm := range vms {
			computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			if i < app.Count {
				vmsBefore[computerName] = struct{}{}
			} else {
				err = u.deleteWorker(ctx, cs, &app, *vm.InstanceID, computerName)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return vmsBefore, nil
}

func (u *simpleUpgrader) waitForNewNodes(ctx context.Context, cs *api.OpenShiftManagedCluster, nodes map[kubeclient.ComputerName]struct{}, ssHashes map[scalesetName]hash) error {
	blob, err := u.readUpdateBlob()
	if err != nil {
		return err
	}

	existingVMs := make(map[instanceName]struct{})
	for _, app := range cs.Properties.AgentPoolProfiles {
		vms, err := u.listVMs(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(app.Name))
		if err != nil {
			return err
		}

		// wait for newly created VMs to reach readiness
		for _, vm := range vms {
			computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			if _, found := nodes[computerName]; !found {
				u.log.Infof("waiting for %s to be ready", computerName)
				err = u.kubeclient.WaitForReadyWorker(ctx, computerName)
				if err != nil {
					return err
				}
				blob[instanceName(*vm.Name)] = ssHashes[ssNameForVM(&vm)]
				if err := u.writeUpdateBlob(blob); err != nil {
					return err
				}
			}
			// store all existing VMs in a map to compare against the VMs
			// stored in the blob in order to clean it up of stale VMs
			existingVMs[instanceName(*vm.Name)] = struct{}{}
		}
	}

	var needsUpdate bool
	for name := range blob {
		if _, ok := existingVMs[name]; !ok {
			delete(blob, name)
			needsUpdate = true
		}
	}
	if needsUpdate {
		return u.writeUpdateBlob(blob)
	}
	return nil
}

func (u *simpleUpgrader) listVMs(ctx context.Context, resourceGroup, scalesetName string) ([]compute.VirtualMachineScaleSetVM, error) {
	vmPages, err := u.vmc.List(ctx, resourceGroup, scalesetName, "", "", "")
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
func (u *simpleUpgrader) updatePlusOne(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, ssHashes map[scalesetName]hash) *api.PluginError {
	// store a list of all the VM instances now, so that if we end up creating
	// new ones (in the crash recovery case, we might not), we can detect which
	// they are
	oldVMs, err := u.listVMs(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(app.Name))
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneListVMs}
	}

	blob, err := u.readUpdateBlob()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneReadBlob}
	}

	// Filter out VMs that do not need to get upgraded. Should speed
	// up retrying failed upgrades.
	oldVMs = u.filterOldVMs(oldVMs, blob, ssHashes)
	vmsBefore := map[string]struct{}{}
	for _, vm := range oldVMs {
		vmsBefore[*vm.InstanceID] = struct{}{}
	}

	for _, vm := range oldVMs {
		ssName := config.GetScalesetName(app.Name)
		u.log.Infof("setting %s capacity to %d", ssName, app.Count+1)
		future, err := u.ssc.Update(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, compute.VirtualMachineScaleSetUpdate{
			Sku: &compute.Sku{
				Capacity: to.Int64Ptr(int64(app.Count) + 1),
			},
		})
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneWaitForReady}
		}

		if err := future.WaitForCompletionRef(ctx, u.ssc.Client()); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneWaitForReady}
		}

		updatedList, err := u.listVMs(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(app.Name))
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
				err = u.kubeclient.WaitForReadyWorker(ctx, computerName)
				if err != nil {
					return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneWaitForReady}
				}
				vmsBefore[*updated.InstanceID] = struct{}{}
				blob[instanceName(*updated.Name)] = ssHashes[ssNameForVM(&updated)]
				if err := u.writeUpdateBlob(blob); err != nil {
					return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneUpdateBlob}
				}
			}
		}

		if err := u.deleteWorker(ctx, cs, app, *vm.InstanceID, kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneDeleteVMs}
		}
		delete(blob, instanceName(*vm.Name))
		if err := u.writeUpdateBlob(blob); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneUpdateBlob}
		}
	}

	return nil
}

func (u *simpleUpgrader) filterOldVMs(vms []compute.VirtualMachineScaleSetVM, blob updateblob, ssHashes map[scalesetName]hash) []compute.VirtualMachineScaleSetVM {
	var oldVMs []compute.VirtualMachineScaleSetVM
	for _, vm := range vms {
		if blob[instanceName(*vm.Name)] != ssHashes[ssNameForVM(&vm)] {
			oldVMs = append(oldVMs, vm)
		} else {
			u.log.Infof("skipping vm %q since it's already updated", *vm.Name)
		}
	}
	return oldVMs
}

func ssNameForVM(vm *compute.VirtualMachineScaleSetVM) scalesetName {
	hostname := strings.Split(*vm.Name, "_")[0]
	return scalesetName(hostname)
}

// updateMasterAgentPool updates one by one all the VMs of the master scale set, in place.
func (u *simpleUpgrader) updateMasterAgentPool(ctx context.Context, cs *api.OpenShiftManagedCluster, ssHashes map[scalesetName]hash) *api.PluginError {
	scaleSetName := config.GetScalesetName(string(api.AgentPoolProfileRoleMaster))

	vms, err := u.listVMs(ctx, cs.Properties.AzProfile.ResourceGroup, scaleSetName)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolListVMs}
	}

	blob, err := u.readUpdateBlob()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReadBlob}
	}

	// range our vms in order, so that if we previously crashed half-way through
	// updating one and it is broken, we pick up where we left off.
	sort.Slice(vms, func(i, j int) bool {
		return *vms[i].VirtualMachineScaleSetVMProperties.OsProfile.ComputerName <
			*vms[j].VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
	})

	vms = u.filterOldVMs(vms, blob, ssHashes)
	for _, vm := range vms {
		computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		u.log.Infof("draining %s", computerName)
		err = u.kubeclient.DeleteMaster(computerName)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDrain}
		}

		{
			u.log.Infof("deallocating %s (%s)", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, *vm.InstanceID)
			future, err := u.vmc.Deallocate(ctx, cs.Properties.AzProfile.ResourceGroup, scaleSetName, *vm.InstanceID)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDeallocate}
			}

			err = future.WaitForCompletionRef(ctx, u.vmc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDeallocate}
			}
		}

		{
			u.log.Infof("updating %s", computerName)
			future, err := u.ssc.UpdateInstances(ctx, cs.Properties.AzProfile.ResourceGroup, scaleSetName, compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
				InstanceIds: &[]string{*vm.InstanceID},
			})
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateVMs}
			}

			err = future.WaitForCompletionRef(ctx, u.ssc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateVMs}
			}
		}

		{
			u.log.Infof("reimaging %s", computerName)
			future, err := u.vmc.Reimage(ctx, cs.Properties.AzProfile.ResourceGroup, scaleSetName, *vm.InstanceID)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReimage}
			}

			err = future.WaitForCompletionRef(ctx, u.vmc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReimage}
			}
		}

		{
			u.log.Infof("starting %s", computerName)
			future, err := u.vmc.Start(ctx, cs.Properties.AzProfile.ResourceGroup, scaleSetName, *vm.InstanceID)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolStart}
			}

			err = future.WaitForCompletionRef(ctx, u.vmc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolStart}
			}
		}

		u.log.Infof("waiting for %s to be ready", computerName)
		err = u.kubeclient.WaitForReadyMaster(ctx, computerName)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolWaitForReady}
		}

		blob[instanceName(*vm.Name)] = ssHashes[ssNameForVM(&vm)]
		if err := u.writeUpdateBlob(blob); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateBlob}
		}
	}

	return nil
}

func (u *simpleUpgrader) deleteWorker(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, instanceID string, nodeName kubeclient.ComputerName) error {
	u.log.Infof("draining %s", nodeName)
	if err := u.kubeclient.DrainAndDeleteWorker(ctx, nodeName); err != nil {
		return err
	}

	u.log.Infof("deleting %s", nodeName)
	future, err := u.vmc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(app.Name), instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, u.vmc.Client())
}

// sortedAgentPoolProfilesForRole returns a shallow copy of the
// AgentPoolProfiles of a given role, sorted by name
func sortedAgentPoolProfilesForRole(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (apps []api.AgentPoolProfile) {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			apps = append(apps, app)
		}
	}

	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })

	return apps
}
