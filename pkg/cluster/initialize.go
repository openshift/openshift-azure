package cluster

import (
	"context"
	"time"

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
		var err error
		u.storageClient, err = storage.NewClient(cs.Config.ConfigStorageAccount, u.storageAccountKey[cs.Config.ConfigStorageAccount], storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
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

func (u *simpleUpgrader) GetStorageAccountKey(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool, accountName string) (string, error) {
	if !isUpdate {
		parameters := azstorage.AccountCreateParameters{
			Sku: &azstorage.Sku{
				Name: azstorage.StandardLRS,
			},
			Kind:     azstorage.Storage,
			Location: &cs.Location,
		}
		if accountName == cs.Config.ConfigStorageAccount {
			parameters.Tags = map[string]*string{
				"type": to.StringPtr("config"),
			}
		}
		future, err := u.accountsClient.Create(ctx, cs.Properties.AzProfile.ResourceGroup, accountName, parameters)
		if err != nil {
			return "", err
		}

		err = future.WaitForCompletionRef(ctx, u.accountsClient.Client())
		if err != nil {
			return "", err
		}
	}

	for found := false; !found; _, found = u.storageAccountKey[accountName] {
		keys, err := u.accountsClient.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, accountName)
		if err != nil {
			return "", err
		}
		if len(*keys.Keys) == 0 {
			// The WaitForCompletionRef above should be enough, but I have seen
			// this return an empty list the first time.
			time.Sleep(1)
			continue
		}
		u.storageAccountKey[accountName] = *(*keys.Keys)[0].Value
	}
	return u.storageAccountKey[accountName], nil
}

func (u *simpleUpgrader) InitializeUpdateBlob(cs *api.OpenShiftManagedCluster, suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range cs.Properties.AgentPoolProfiles {
		h, err := u.hasher.HashScaleSet(cs, &app, u.storageAccountKey)
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
