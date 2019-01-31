package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// Initialize does the following:
// - ensures the storageClient is initialised (this is dependent on the config
//   storage account existing, which is why it can't be done before)
// - ensures the expected containers (config, etcd, update) exist
// - populates the config blob
func (u *simpleUpgrader) Initialize(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	if u.storageClient == nil {
		keys, err := u.accountsClient.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
		if err != nil {
			return err
		}
		u.storageClient, err = storage.NewClient(cs.Config.ConfigStorageAccount, *(*keys.Keys)[0].Value, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
		if err != nil {
			return err
		}
	}
	bsc := u.storageClient.GetBlobService()

	// etcd data container
	c := bsc.GetContainerReference(EtcdBackupContainerName)
	_, err := c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	u.updateBlobService, err = updateblob.NewBlobService(bsc)
	return err
}

func (u *simpleUpgrader) InitializeUpdateBlob(cs *api.OpenShiftManagedCluster, suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range cs.Properties.AgentPoolProfiles {
		h, err := u.hasher.HashScaleSet(cs, &app)
		if err != nil {
			return err
		}
		if app.Role == api.AgentPoolProfileRoleMaster {
			for i := int64(0); i < app.Count; i++ {
				name := config.GetMasterInstanceName(i)
				blob.InstanceHashes[name] = h
			}
		} else {
			blob.ScalesetHashes[config.GetScalesetName(&app, suffix)] = h
		}
	}
	return u.updateBlobService.Write(blob)
}
