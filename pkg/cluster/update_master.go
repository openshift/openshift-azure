package cluster

import (
	"bytes"
	"context"
	"sort"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/config"
)

func (u *simpleUpgrader) filterOldVMs(vms []compute.VirtualMachineScaleSetVM, blob *updateblob, ssHash []byte) []compute.VirtualMachineScaleSetVM {
	var oldVMs []compute.VirtualMachineScaleSetVM
	for _, vm := range vms {
		if !bytes.Equal(blob.InstanceHashes[instanceName(*vm.Name)], ssHash) {
			oldVMs = append(oldVMs, vm)
		} else {
			u.log.Infof("skipping vm %q since it's already updated", *vm.Name)
		}
	}
	return oldVMs
}

// updateMasterAgentPool updates one by one all the VMs of the master scale set, in place.
func (u *simpleUpgrader) updateMasterAgentPool(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) *api.PluginError {
	ssName := config.GetScalesetName(app.Name)
	ssHash, err := u.hasher.HashScaleSet(cs, app)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolHashScaleSet}
	}

	blob, err := u.readUpdateBlob()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReadBlob}
	}

	vms, err := u.listVMs(ctx, cs.Properties.AzProfile.ResourceGroup, ssName)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolListVMs}
	}

	// range our vms in order, so that if we previously crashed half-way through
	// updating one and it is broken, we pick up where we left off.
	sort.Slice(vms, func(i, j int) bool {
		return *vms[i].VirtualMachineScaleSetVMProperties.OsProfile.ComputerName <
			*vms[j].VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
	})

	vms = u.filterOldVMs(vms, blob, ssHash)
	for _, vm := range vms {
		computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		u.log.Infof("draining %s", computerName)
		err = u.kubeclient.DeleteMaster(computerName)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDrain}
		}

		{
			u.log.Infof("deallocating %s (%s)", *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName, *vm.InstanceID)
			future, err := u.vmc.Deallocate(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, *vm.InstanceID)
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
			future, err := u.ssc.UpdateInstances(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
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
			future, err := u.vmc.Reimage(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, *vm.InstanceID)
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
			future, err := u.vmc.Start(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, *vm.InstanceID)
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

		blob.InstanceHashes[instanceName(*vm.Name)] = ssHash
		if err := u.writeUpdateBlob(blob); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateBlob}
		}
	}

	return nil
}
