package cluster

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// initialize does the following:
// - ensures the storageClient is initialised (this is dependent on the config
//   storage account existing, which is why it can't be done before)
// - ensures the expected containers (config, etcd, update) exist
// - populates the config blob
func (u *simpleUpgrader) initialize(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
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

	// update tracking container
	u.updateContainer = bsc.GetContainerReference(updateContainerName)
	_, err = u.updateContainer.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	// cluster config container
	c = bsc.GetContainerReference(ConfigContainerName)
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	b := c.GetBlobReference(ConfigBlobName)

	csj, err := json.Marshal(cs)
	if err != nil {
		return err
	}

	return b.CreateBlockBlobFromReader(bytes.NewReader(csj), nil)
}
