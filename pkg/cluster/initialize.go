package cluster

import (
	"bytes"
	"context"
	"encoding/json"

	azstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

func (u *Upgrade) initializeStorageClients(ctx context.Context) error {
	if u.StorageClient == nil {
		if u.Cs.Config.ConfigStorageAccountKey == "" {
			keys, err := u.AccountsClient.ListKeys(ctx, u.Cs.Properties.AzProfile.ResourceGroup, u.Cs.Config.ConfigStorageAccount)
			if err != nil {
				return err
			}
			u.Cs.Config.ConfigStorageAccountKey = *(*keys.Keys)[0].Value
		}

		var err error
		u.StorageClient, err = storage.NewClient(u.Log, u.Cs.Config.ConfigStorageAccount, u.Cs.Config.ConfigStorageAccountKey, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
		if err != nil {
			return err
		}

		bsc := u.StorageClient.GetBlobService()
		u.UpdateBlobService = updateblob.NewBlobService(bsc)
	}

	return nil
}

func (u *Upgrade) writeBlob(blobName string, cs *api.OpenShiftManagedCluster) error {
	bsc := u.StorageClient.GetBlobService()
	c := bsc.GetContainerReference(ConfigContainerName)
	b := c.GetBlobReference(blobName)

	json, err := json.Marshal(cs)
	if err != nil {
		return err
	}

	return b.CreateBlockBlobFromReader(bytes.NewReader(json), nil)
}

// WriteStartupBlobs writes the blobs to the SA for all agent pool profiles
func (u *Upgrade) WriteStartupBlobs() error {
	u.Log.Info("writing startup blobs")
	err := u.writeBlob(MasterStartupBlobName, u.Cs)
	if err != nil {
		return err
	}
	sup, err := startup.New(u.Log, u.Cs, u.TestConfig)
	if err != nil {
		return err
	}
	return u.writeBlob(WorkerStartupBlobName, sup.GetWorkerCs())
}

// CreateOrUpdateConfigStorageAccount creates a new storage account for config if missing
func (u *Upgrade) CreateOrUpdateConfigStorageAccount(ctx context.Context) error {
	u.Log.Info("creating/updating storage account")

	err := u.AccountsClient.Create(ctx, u.Cs.Properties.AzProfile.ResourceGroup, u.Cs.Config.ConfigStorageAccount, azstorage.AccountCreateParameters{
		Sku: &azstorage.Sku{
			Name: azstorage.StandardLRS,
		},
		Kind:     azstorage.Storage,
		Location: &u.Cs.Location,
		Tags: map[string]*string{
			"type": to.StringPtr("config"),
		},
		AccountPropertiesCreateParameters: &azstorage.AccountPropertiesCreateParameters{
			EnableHTTPSTrafficOnly: to.BoolPtr(true),
		},
	})
	if err != nil {
		return err
	}

	err = u.initializeStorageClients(ctx)
	if err != nil {
		return err
	}

	bsc := u.StorageClient.GetBlobService()

	// cluster config container
	c := bsc.GetContainerReference(ConfigContainerName)
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	// etcd data container
	c = bsc.GetContainerReference(EtcdBackupContainerName)
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	// update container
	c = bsc.GetContainerReference(updateblob.UpdateContainerName)
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	return nil
}

// InitializeUpdateBlob hashes the cluster values and stores them in blobs
func (u *Upgrade) InitializeUpdateBlob(suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range u.Cs.Properties.AgentPoolProfiles {
		h, err := u.Hasher.HashScaleSet(u.Cs, &app)
		if err != nil {
			return err
		}

		switch app.Role {
		case api.AgentPoolProfileRoleMaster:
			for i := int64(0); i < app.Count; i++ {
				hostname := names.GetHostname(&app, suffix, i)
				blob.HostnameHashes[hostname] = h
			}

		default:
			blob.ScalesetHashes[names.GetScalesetName(&app, suffix)] = h
		}
	}
	return u.UpdateBlobService.Write(blob)
}

// ResetUpdateBlob resets the update blob to its initial (default) state
func (u *Upgrade) ResetUpdateBlob() error {
	blob := updateblob.NewUpdateBlob()
	return u.UpdateBlobService.Write(blob)
}
