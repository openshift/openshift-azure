package upgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

func TestInitializeCluster(t *testing.T) {
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
	bsa.EXPECT().GetContainerReference("etcd").Return(etcdCr)
	etcdCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	configCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference("config").Return(configCr)
	configCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	configBlob := mock_storage.NewMockBlob(gmc)
	configCr.EXPECT().GetBlobReference("config").Return(configBlob)

	cs := fixtures.NewTestOpenShiftCluster()
	csj, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}
	var finalError error
	configBlob.EXPECT().CreateBlockBlobFromReader(bytes.NewReader(csj), nil).Return(finalError)

	log.New(logrus.NewEntry(logrus.New()))
	if err := si.InitializeCluster(context.Background(), cs); err != finalError {
		t.Errorf("simpleInitializer.InitializeCluster() error = %v", err)
	}
}
