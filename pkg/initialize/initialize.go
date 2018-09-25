package initialize

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/azure"
	"github.com/openshift/openshift-azure/pkg/log"
)

type Initializer interface {
	InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error
}

type simpleInitializer struct{}

var _ Initializer = &simpleInitializer{}

func NewSimpleInitializer(entry *logrus.Entry) Initializer {
	log.New(entry)
	return &simpleInitializer{}
}

func (*simpleInitializer) InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	saClient, err := azure.NewAccountStorageClient(ctx, ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string), cs.Properties.AzProfile.SubscriptionID)
	if err != nil {
		return err
	}
	storageAcc, err := saClient.GetStorageAccount(ctx, cs.Properties.AzProfile.ResourceGroup, "config")
	if err != nil {
		return err
	}
	sClient, err := azure.NewStorageClient(ctx, storageAcc["key"], storageAcc["name"])
	if err != nil {
		return err
	}

	// etcd data container
	c := sClient.GetContainerReference("etcd")
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	// cluster config container
	c = sClient.GetContainerReference("config")
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
