package cluster

import (
	"bytes"
	"context"
	"encoding/json"

	azstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

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

func (u *simpleUpgrader) CreateConfigStorageAccount(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	parameters := azstorage.AccountCreateParameters{
		Sku: &azstorage.Sku{
			Name: azstorage.StandardLRS,
		},
		Kind:     azstorage.Storage,
		Location: &cs.Location,
	}
	parameters.Tags = map[string]*string{
		"type": to.StringPtr("config"),
	}
	return u.accountsClient.Create(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount, parameters)
}

func (u *simpleUpgrader) InitializeUpdateBlob(cs *api.OpenShiftManagedCluster, suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range cs.Properties.AgentPoolProfiles {
		switch app.Role {
		case api.AgentPoolProfileRoleMaster:
			h, err := u.hasher.HashMasterScaleSet(cs, &app)
			if err != nil {
				return err
			}
			for i := int64(0); i < app.Count; i++ {
				hostname := config.GetHostname(&app, suffix, i)
				blob.HostnameHashes[hostname] = h
			}

		default:
			h, err := u.hasher.HashWorkerScaleSet(cs, &app)
			if err != nil {
				return err
			}
			blob.ScalesetHashes[config.GetScalesetName(&app, suffix)] = h
		}
	}
	return u.updateBlobService.Write(blob)
}

func (u *simpleUpgrader) ResetUpdateBlob(cs *api.OpenShiftManagedCluster) error {
	blob := updateblob.NewUpdateBlob()
	return u.updateBlobService.Write(blob)
}
