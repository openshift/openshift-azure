package initialize

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
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
	az, err := newAzureClients(ctx, cs)
	if err != nil {
		return err
	}

	bsc := az.storage.GetBlobService()

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
