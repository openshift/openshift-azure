package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_updatehash"
)

func TestInitialize(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	storageClient := mock_storage.NewMockClient(gmc)
	uh := mock_updatehash.NewMockUpdateHash(gmc)
	u := &simpleUpgrader{
		storageClient: storageClient,
		updateHash:    uh,
	}
	bsa := mock_storage.NewMockBlobStorageClient(gmc)
	storageClient.EXPECT().GetBlobService().Return(bsa)

	etcdCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(EtcdBackupContainerName).Return(etcdCr)
	etcdCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	updateCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(updateContainerName).Return(updateCr)
	uh.EXPECT().SetContainer(updateCr)

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

	if err := u.initialize(context.Background(), cs); err != finalError {
		t.Errorf("simpleUpgrader.initialize() error = %v", err)
	}
}
