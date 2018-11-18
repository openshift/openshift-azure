package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

func TestInitialize(t *testing.T) {
	gmc := gomock.NewController(t)
	accountsClient := mock_azureclient.NewMockAccountsClient(gmc)
	storageClient := mock_storage.NewMockClient(gmc)
	si := &simpleUpgrader{
		pluginConfig:   api.PluginConfig{},
		accountsClient: accountsClient,
		storageClient:  storageClient,
	}
	bsa := mock_storage.NewMockBlobStorageClient(gmc)
	storageClient.EXPECT().GetBlobService().Return(bsa)

	etcdCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(EtcdBackupContainerName).Return(etcdCr)
	etcdCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	updateCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference("update").Return(updateCr)
	updateCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	configCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(ConfigContainerName).Return(configCr)
	configCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	configBlob := mock_storage.NewMockBlob(gmc)
	configCr.EXPECT().GetBlobReference(ConfigBlobName).Return(configBlob)

	cs := &api.OpenShiftManagedCluster{}
	csj, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}
	var finalError error
	configBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader(csj), nil).Return(finalError)

	log.New(logrus.NewEntry(logrus.New()))
	if err := si.initialize(context.Background(), cs); err != finalError {
		t.Errorf("simpleUpgrader.initialize() error = %v", err)
	}
}
