package cluster

import (
	"bytes"
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
)

// findScaleSets discovers all the scalesets that exist for a given agent pool.
// The first scaleset which matches desiredHash, if one exists, is denoted the
// "target".  We will work to get all our VMs running there.  Any other
// scalesets are "sources".  We will work to get rid of VMs running in these.
func (u *Upgrade) findScaleSets(ctx context.Context, resourceGroup string, app *api.AgentPoolProfile, blob *updateblob.UpdateBlob, desiredHash []byte) (*compute.VirtualMachineScaleSet, []compute.VirtualMachineScaleSet, error) {
	scalesets, err := u.Ssc.List(ctx, resourceGroup)
	if err != nil {
		return nil, nil, err
	}

	var target *compute.VirtualMachineScaleSet
	var sources []compute.VirtualMachineScaleSet

	prefix := names.GetScalesetName(app, "")

	for i, ss := range scalesets {
		if !strings.HasPrefix(*ss.Name, prefix) {
			continue
		}

		// Note: we consult the blob to discover the persisted scaleset hash,
		// rather than recalculating it on the fly.  This is because Kubernetes
		// may have changed the scaleset object after we created it.  We
		// consider any such changes irrelevant to our hashing scheme.  For any
		// worker scale set, the scale set hash persisted in the blob is
		// expected to be immutable.
		if target == nil && bytes.Equal(blob.ScalesetHashes[*ss.Name], desiredHash) {
			u.Log.Infof("found target scaleset %s", *ss.Name)
			target = &scalesets[i]

		} else {
			u.Log.Infof("found source scaleset %s", *ss.Name)
			sources = append(sources, ss)
		}
	}

	return target, sources, nil
}

// updateWorkerAgentPool updates one by one all the VMs of a worker agent pool.
// It defines a "target" scale set, which is known to be up-to-date because its
// hash matches desiredHash.  The goal is for the correct number of instances to
// be running in the "target" scale set.  In update scenarios, there will be a
// "source" scale set which contains out-of-date instances (in crash recovery
// scenarios, there could be multiple of these).
func (u *Upgrade) UpdateWorkerAgentPool(ctx context.Context, app *api.AgentPoolProfile, suffix string) *api.PluginError {
	u.Log.Infof("updating worker agent pool %s", app.Name)

	desiredHash, err := u.Hasher.HashScaleSet(u.Cs, app)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolHashScaleSet}
	}

	blob, err := u.UpdateBlobService.Read()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolReadBlob}
	}

	target, sources, err := u.findScaleSets(ctx, u.Cs.Properties.AzProfile.ResourceGroup, app, blob, desiredHash)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolListScaleSets}
	}

	if target == nil {
		// No pre-existing scaleset exists which matches desiredHash.  Create a
		// new zero instance scaleset to be our target.  Clean scales should not
		// hit this codepath.
		var err *api.PluginError
		target, err = u.createWorkerScaleSet(ctx, app, suffix, blob)
		if err != nil {
			return err
		}
	}

	targetScaler := u.ScalerFactory.New(u.Log, u.Ssc, u.Vmc, u.Interface, u.Cs.Properties.AzProfile.ResourceGroup, target, u.TestConfig)

	// One by one, get rid of instances in any "source" scalesets.  Clean scales
	// should not hit this codepath.
	for _, source := range sources {
		sourceScaler := u.ScalerFactory.New(u.Log, u.Ssc, u.Vmc, u.Interface, u.Cs.Properties.AzProfile.ResourceGroup, &source, u.TestConfig)

		for *source.Sku.Capacity > 0 {
			if *target.Sku.Capacity < app.Count {
				if err := targetScaler.Scale(ctx, *target.Sku.Capacity+1); err != nil {
					return err
				}
			}

			if err := sourceScaler.Scale(ctx, *source.Sku.Capacity-1); err != nil {
				return err
			}
		}

		if err := u.deleteWorkerScaleSet(ctx, blob, &source, u.Cs.Properties.AzProfile.ResourceGroup); err != nil {
			return err
		}
	}

	// Finally, ensure our "target" scaleset is the right size.
	return targetScaler.Scale(ctx, app.Count)
}

// createWorkerScaleSet creates a new scaleset to be our target.  For now, for
// simplicity, the scaleset has zero instances - we fix this up later.  TODO:
// improve this.
func (u *Upgrade) createWorkerScaleSet(ctx context.Context, app *api.AgentPoolProfile, suffix string, blob *updateblob.UpdateBlob) (*compute.VirtualMachineScaleSet, *api.PluginError) {
	hash, err := u.Hasher.HashScaleSet(u.Cs, app)
	if err != nil {
		return nil, &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolHashScaleSet}
	}

	target, err := u.Arm.Vmss(app, "", suffix)
	if err != nil {
		return nil, &api.PluginError{Err: err, Step: api.PluginStepGenerateARM}
	}
	target.Sku.Capacity = to.Int64Ptr(0)

	u.Log.Infof("creating target scaleset %s", names.GetScalesetName(app, suffix))
	err = u.Ssc.CreateOrUpdate(ctx, u.Cs.Properties.AzProfile.ResourceGroup, *target.Name, *target)
	if err != nil {
		return nil, &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolCreateScaleSet}
	}

	// Persist the scaleset's hash: this is expected to be immutable for the
	// lifetime of the scaleset.  We do this *after* the scaleset is
	// successfully created to avoid leaking blob entries.
	blob.ScalesetHashes[*target.Name] = hash
	if err = u.UpdateBlobService.Write(blob); err != nil {
		return nil, &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolUpdateBlob}
	}

	return target, nil
}

// deleteWorkerScaleSet deletes a (presumably empty) scaleset.
func (u *Upgrade) deleteWorkerScaleSet(ctx context.Context, blob *updateblob.UpdateBlob, ss *compute.VirtualMachineScaleSet, resourceGroup string) *api.PluginError {
	// Delete the persisted scaleset hash.  We do this *before* the scaleset is
	// deleted to avoid leaking blob entries.
	delete(blob.ScalesetHashes, *ss.Name)
	if err := u.UpdateBlobService.Write(blob); err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolUpdateBlob}
	}

	u.Log.Infof("deleting scaleset %s", *ss.Name)
	err := u.Ssc.Delete(ctx, resourceGroup, *ss.Name)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolDeleteScaleSet}
	}

	return nil
}
