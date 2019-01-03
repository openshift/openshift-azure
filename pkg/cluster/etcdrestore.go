package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// TODO: Evacuate is solely used by the etcd restore code.  It should probably
// not be exactly here.
func (u *simpleUpgrader) Evacuate(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	// We may need/want to delete all the scalesets in the future
	future, err := u.ssc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, config.MasterScalesetName)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepScaleSetDelete}
	}
	err = future.WaitForCompletionRef(ctx, u.ssc.Client())
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepScaleSetDelete}
	}
	// TODO: this code should be rolled into initialize()
	if u.storageClient == nil {
		keys, err := u.accountsClient.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
		}
		u.storageClient, err = storage.NewClient(cs.Config.ConfigStorageAccount, *(*keys.Keys)[0].Value, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
		}
	}
	bsc := u.storageClient.GetBlobService()
	u.updateBlobService, err = updateblob.NewBlobService(bsc)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}
	err = u.updateBlobService.Write(updateblob.NewUpdateBlob())
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
	}
	return nil
}
