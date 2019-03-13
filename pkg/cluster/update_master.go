package cluster

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
)

// UpdateMasterAgentPool updates one by one all the VMs of the master scale set,
// in place.
func (u *simpleUpgrader) UpdateMasterAgentPool(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) *api.PluginError {
	ssName := config.MasterScalesetName

	blob, err := u.updateBlobService.Read()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReadBlob}
	}

	for i := int64(0); i < app.Count; i++ {
		desiredHash, err := u.hasher.HashMasterScaleSet(cs, app, i)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolHashMasterScaleSet}
		}

		hostname := config.GetHostname(app, "", i)
		instanceID := fmt.Sprintf("%d", i)

		if bytes.Equal(blob.HostnameHashes[hostname], desiredHash) {
			u.log.Infof("skipping vm %q since it's already updated", hostname)
			continue
		}

		u.log.Infof("draining %s", hostname)
		err = u.kubeclient.DeleteMaster(hostname)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDrain}
		}

		u.log.Infof("deallocating %s", hostname)
		err = u.vmc.Deallocate(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, instanceID)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDeallocate}
		}

		u.log.Infof("updating %s", hostname)
		err = u.ssc.UpdateInstances(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
			InstanceIds: &[]string{instanceID},
		})
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateVMs}
		}

		u.log.Infof("reimaging %s", hostname)
		err = u.vmc.Reimage(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, instanceID, nil)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReimage}
		}

		u.log.Infof("starting %s", hostname)
		err = u.vmc.Start(ctx, cs.Properties.AzProfile.ResourceGroup, ssName, instanceID)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolStart}
		}

		u.log.Infof("waiting for %s to be ready", hostname)
		err = u.kubeclient.WaitForReadyMaster(ctx, hostname)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolWaitForReady}
		}

		blob.HostnameHashes[hostname] = desiredHash
		if err := u.updateBlobService.Write(blob); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateBlob}
		}
	}

	return nil
}
