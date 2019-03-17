package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
)

// CreateOrUpdateSyncPod creates or updates the sync pod.
func (u *simpleUpgrader) CreateOrUpdateSyncPod(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	u.log.Infof("updating sync pod")

	err := u.writeBlob(SyncBlobName, cs)
	if err != nil {
		return err
	}

	hash, err := u.hasher.HashSyncPod(cs)
	if err != nil {
		return err
	}

	return u.Kubeclient.EnsureSyncPod(ctx, cs.Config.Images.Sync, hash)
}
