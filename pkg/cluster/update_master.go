package cluster

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
)

// UpdateMasterAgentPool updates one by one all the VMs of the master scale set,
// in place.
func (u *Upgrade) UpdateMasterAgentPool(ctx context.Context, app *api.AgentPoolProfile) *api.PluginError {
	ssName := names.MasterScalesetName

	blob, err := u.UpdateBlobService.Read()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReadBlob}
	}

	desiredHash, err := u.Hasher.HashScaleSet(u.Cs, app)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolHashScaleSet}
	}

	for i := int64(0); i < app.Count; i++ {
		hostname := names.GetHostname(app, "", i)
		instanceID := fmt.Sprintf("%d", i)

		if bytes.Equal(blob.HostnameHashes[hostname], desiredHash) {
			u.Log.Infof("skipping vm %q since it's already updated", hostname)
			continue
		}

		u.Log.Infof("draining %s", hostname)
		err = u.Interface.DeleteMaster(hostname)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDrain}
		}

		u.Log.Infof("deallocating %s", hostname)
		err = u.Vmc.Deallocate(ctx, u.Cs.Properties.AzProfile.ResourceGroup, ssName, instanceID)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolDeallocate}
		}

		u.Log.Infof("updating %s", hostname)
		err = u.Ssc.UpdateInstances(ctx, u.Cs.Properties.AzProfile.ResourceGroup, ssName, compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
			InstanceIds: &[]string{instanceID},
		})
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateVMs}
		}

		u.Log.Infof("reimaging %s", hostname)
		err = u.Vmc.Reimage(ctx, u.Cs.Properties.AzProfile.ResourceGroup, ssName, instanceID, nil)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolReimage}
		}

		u.Log.Infof("starting %s", hostname)
		err = u.Vmc.Start(ctx, u.Cs.Properties.AzProfile.ResourceGroup, ssName, instanceID)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolStart}
		}

		u.Log.Infof("waiting for %s to be ready", hostname)
		err = u.Interface.WaitForReadyMaster(ctx, hostname)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolWaitForReady}
		}

		blob.HostnameHashes[hostname] = desiredHash

		if err := u.UpdateBlobService.Write(blob); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateMasterAgentPoolUpdateBlob}
		}
	}

	return nil
}
