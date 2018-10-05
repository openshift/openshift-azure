package upgrade

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

func (si *simpleUpgrader) InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}

	accounts := azureclient.NewAccountsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, si.pluginConfig.AcceptLanguages)
	keys, err := accounts.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
	if err != nil {
		return err
	}

	storageClient, err := storage.NewClient(cs.Config.ConfigStorageAccount, *(*keys.Keys)[0].Value, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
	if err != nil {
		return err
	}

	bsc := storageClient.GetBlobService()

	// etcd data container
	c := bsc.GetContainerReference("etcd")
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	// cluster config container
	c = bsc.GetContainerReference("config")
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	b := c.GetBlobReference("config")

	csj, err := json.Marshal(cs)
	if err != nil {
		return err
	}

	return b.CreateBlockBlobFromReader(bytes.NewReader(csj), nil)
}
