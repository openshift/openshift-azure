package cluster

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient/mock_storage"
)

func TestInitialize(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()
	storageClient := mock_storage.NewMockClient(gmc)
	u := &simpleUpgrader{
		storageClient: storageClient,
	}
	bsa := mock_storage.NewMockBlobStorageClient(gmc)
	storageClient.EXPECT().GetBlobService().Return(bsa)

	etcdCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(EtcdBackupContainerName).Return(etcdCr)
	etcdCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)

	updateCr := mock_storage.NewMockContainer(gmc)
	bsa.EXPECT().GetContainerReference(updateblob.UpdateContainerName).Return(updateCr)
	updateCr.EXPECT().CreateIfNotExists(nil).Return(true, nil)
	cs := &api.OpenShiftManagedCluster{}

	if err := u.Initialize(context.Background(), cs); err != nil {
		t.Errorf("simpleUpgrader.Initialize() error = %v", err)
	}
}
