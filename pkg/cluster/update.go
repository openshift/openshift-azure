package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) *api.PluginError {
	err := u.createClients(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}
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
	ssHashes, err := hashScaleSets(azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepHashScaleSets}
	}
	err = managedcluster.WaitForHealthz(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	err = u.waitForNewNodes(ctx, cs, vmsBefore, ssHashes)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
	}
	if perr := u.updateInPlace(ctx, cs, api.AgentPoolProfileRoleMaster, ssHashes); perr != nil {
		return perr
	}
	if perr := u.updatePlusOne(ctx, cs, api.AgentPoolProfileRoleInfra, ssHashes); perr != nil {
		return perr
	}
	if perr := u.updatePlusOne(ctx, cs, api.AgentPoolProfileRoleCompute, ssHashes); perr != nil {
		return perr
	}
	return nil
}

func (u *simpleUpgrader) getNodesAndDrain(ctx context.Context, cs *api.OpenShiftManagedCluster) (map[string]struct{}, error) {
	vmsBefore := map[string]struct{}{}

	for _, agent := range cs.Properties.AgentPoolProfiles {
		vms, err := listVMs(ctx, cs, u.vmc, agent.Role)
		if err != nil {
			return nil, err
		}

		for i, vm := range vms {
			if i < agent.Count {
				vmsBefore[*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName] = struct{}{}
			} else {
				err = u.delete(ctx, cs, agent.Role, *vm.InstanceID, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return vmsBefore, nil
}

func (u *simpleUpgrader) waitForNewNodes(ctx context.Context, cs *api.OpenShiftManagedCluster, nodes map[string]struct{}, ssHashes map[scalesetName]hash) error {
	blob, err := u.readBlob()
	if err != nil {
		return err
	}

	existingVMs := make(map[instanceName]struct{})
	for _, agent := range cs.Properties.AgentPoolProfiles {
		vms, err := listVMs(ctx, cs, u.vmc, agent.Role)
		if err != nil {
			return err
		}

		// wait for newly created VMs to reach readiness
		for _, vm := range vms {
			hostname := *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
			if _, found := nodes[hostname]; !found {
				log.Infof("waiting for %s to be ready", hostname)
				err = waitForReady(ctx, cs, agent.Role, hostname, u.kubeclient)
				if err != nil {
					return err
				}
				blob[instanceName(*vm.Name)] = ssHashes[scalesetName(agent.Name)]
				if err := u.updateBlob(blob); err != nil {
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
		return u.updateBlob(blob)
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

func listVMs(ctx context.Context, cs *api.OpenShiftManagedCluster, vmc azureclient.VirtualMachineScaleSetVMsClient, role api.AgentPoolProfileRole) ([]compute.VirtualMachineScaleSetVM, error) {
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

// updatePlusOne creates new VMs and removes old VMs one by one.
func (u *simpleUpgrader) updatePlusOne(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, ssHashes map[scalesetName]hash) *api.PluginError {
	count := getCount(cs, role)

	// store a list of all the VM instances now, so that if we end up creating
	// new ones (in the crash recovery case, we might not), we can detect which
	// they are
	oldVMs, err := listVMs(ctx, cs, u.vmc, role)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneListVMs}
	}

	blob, err := u.readBlob()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneReadBlob}
	}

	// Filter out VMs that do not need to get upgraded. Should speed
	// up retrying failed upgrades.
	oldVMs = filterOldVMs(oldVMs, blob, ssHashes)
	vmsBefore := map[string]struct{}{}
	for _, vm := range oldVMs {
		vmsBefore[*vm.InstanceID] = struct{}{}
	}

	for _, vm := range oldVMs {
		log.Infof("setting ss-%s capacity to %d", role, count+1)
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

		updatedList, err := listVMs(ctx, cs, u.vmc, role)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneListVMs}
		}

		// wait for newly created VMs to reach readiness (n.b. one alternative to
		// this approach would be for the CSE to not return until the node is
		// ready, but that is also problematic)
		for _, updated := range updatedList {
			if _, found := vmsBefore[*updated.InstanceID]; !found {
				log.Infof("waiting for %s to be ready", *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
				err = waitForReady(ctx, cs, role, *updated.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, u.kubeclient)
				if err != nil {
					return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneWaitForReady}
				}
				vmsBefore[*updated.InstanceID] = struct{}{}
				blob[instanceName(*updated.Name)] = ssHashes[ssNameForVm(&updated)]
				if err := u.updateBlob(blob); err != nil {
					return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneUpdateBlob}
				}
			}
		}

		if err := u.delete(ctx, cs, role, *vm.InstanceID, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneDeleteVMs}
		}
		delete(blob, instanceName(*vm.Name))
		if err := u.updateBlob(blob); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdatePlusOneUpdateBlob}
		}
	}

	return nil
}

func filterOldVMs(vms []compute.VirtualMachineScaleSetVM, blob map[instanceName]hash, ssHashes map[scalesetName]hash) []compute.VirtualMachineScaleSetVM {
	var oldVMs []compute.VirtualMachineScaleSetVM
	for _, vm := range vms {
		if blob[instanceName(*vm.Name)] != ssHashes[ssNameForVm(&vm)] {
			oldVMs = append(oldVMs, vm)
		} else {
			log.Infof("skipping vm %q since it's already updated", *vm.Name)
		}
	}
	return oldVMs
}

func ssNameForVm(vm *compute.VirtualMachineScaleSetVM) scalesetName {
	hostname := strings.Split(*vm.Name, "_")[0]
	return scalesetName(hostname)
}

// updateInPlace updates one by one all the VMs of a scale set, in place.
func (u *simpleUpgrader) updateInPlace(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, ssHashes map[scalesetName]hash) *api.PluginError {
	vms, err := listVMs(ctx, cs, u.vmc, role)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceListVMs}
	}

	sorted, err := sortMasterVMsByHealth(vms, cs, u.kubeclient)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceSortMasters}
	}

	blob, err := u.readBlob()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceReadBlob}
	}

	sorted = filterOldVMs(sorted, blob, ssHashes)
	for _, vm := range sorted {
		log.Infof("draining %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		err = u.drain(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceDrain}
		}

		{
			log.Infof("deallocating %s (%s)", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, *vm.InstanceID)
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
			log.Infof("updating %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
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
			log.Infof("reimaging %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
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
			log.Infof("starting %s", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			future, err := u.vmc.Start(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), *vm.InstanceID)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceStart}
			}

			err = future.WaitForCompletionRef(ctx, u.vmc.Client())
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceStart}
			}
		}

		log.Infof("waiting for %s to be ready", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		err = waitForReady(ctx, cs, role, *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, u.kubeclient)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceWaitForReady}
		}

		blob[instanceName(*vm.Name)] = ssHashes["master"]
		if err := u.updateBlob(blob); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateInPlaceUpdateBlob}
		}
	}

	return nil
}

func sortMasterVMsByHealth(vms []compute.VirtualMachineScaleSetVM, cs *api.OpenShiftManagedCluster, kc kubernetes.Interface) ([]compute.VirtualMachineScaleSetVM, error) {
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

func (u *simpleUpgrader) delete(ctx context.Context, cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, instanceID, nodeName string) error {
	log.Infof("draining %s", nodeName)
	if err := u.drain(ctx, cs, role, nodeName); err != nil {
		return err
	}

	log.Infof("deleting %s", nodeName)
	future, err := u.vmc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-"+string(role), instanceID)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, u.vmc.Client())
}
