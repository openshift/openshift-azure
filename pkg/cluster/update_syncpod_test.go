package cluster

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_cluster"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_updateblob"
)

func TestUpdateSyncPod(t *testing.T) {
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	ctx := context.Background()
	cs := &api.OpenShiftManagedCluster{}

	ubs := mock_updateblob.NewMockBlobService(gmc)
	kc := mock_kubeclient.NewMockKubeclient(gmc)
	hasher := mock_cluster.NewMockHasher(gmc)

	u := &simpleUpgrader{
		updateBlobService: ubs,
		Kubeclient:        kc,
		log:               logrus.NewEntry(logrus.StandardLogger()),
		hasher:            hasher,
	}

	c := ubs.EXPECT().Read().Return(updateblob.NewUpdateBlob(), nil)
	c = hasher.EXPECT().HashSyncPod(cs).Return([]byte("updated"), nil).After(c)
	c = kc.EXPECT().DeletePod(ctx, "kube-system", "sync-master-000000").Return(nil).After(c)
	c = kc.EXPECT().WaitForReadySyncPod(ctx).Return(nil).After(c)

	uBlob := updateblob.NewUpdateBlob()
	uBlob.SyncPodHash = []byte("updated")
	c = ubs.EXPECT().Write(uBlob).Return(nil).After(c)

	if err := u.UpdateSyncPod(ctx, cs); err != nil {
		t.Errorf("simpleUpgrader.updateSyncPod() = %v", err)
	}
}
