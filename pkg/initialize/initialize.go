package initialize

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type Initializer interface {
	InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error
}

type simpleInitializer struct {
	pluginConfig api.PluginConfig
}

var _ Initializer = &simpleInitializer{}

// NewSimpleInitializer create a new Initilizer
func NewSimpleInitializer(entry *logrus.Entry, pluginConfig api.PluginConfig) Initializer {
	log.New(entry)
	return &simpleInitializer{pluginConfig: pluginConfig}
}

func (si *simpleInitializer) InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	az, err := azureclient.NewAzureClients(ctx, cs, si.pluginConfig)
	if err != nil {
		return err
	}

	keys, err := az.Accounts.ListKeys(context.Background(), cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
	if err != nil {
		return err
	}

	var storageClient storage.Client
	storageClient, err = storage.NewClient(cs.Config.ConfigStorageAccount, *(*keys.Keys)[0].Value, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
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
