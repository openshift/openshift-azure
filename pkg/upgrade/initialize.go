package upgrade

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func (si *simpleUpgrader) InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	authorizer, err := azureclient.NewAuthorizerFromCtx(ctx)
	if err != nil {
		return err
	}

	accountClient := azureclient.NewAccountsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, si.pluginConfig)
	keys, err := accountClient.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
	if err != nil {
		return err
	}

	storageClient, err := azureclient.NewStorageClient(cs.Config.ConfigStorageAccount, *(*keys.Keys)[0].Value)
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
