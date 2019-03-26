package cluster

import (
	"context"
)

// CreateOrUpdateSyncPod creates or updates the sync pod.
func (u *simpleUpgrader) CreateOrUpdateSyncPod(ctx context.Context) error {
	u.log.Infof("updating sync pod")

	err := u.writeBlob(SyncBlobName, u.cs)
	if err != nil {
		return err
	}

	hash, err := u.hasher.HashSyncPod(u.cs)
	if err != nil {
		return err
	}

	return u.Kubeclient.EnsureSyncPod(ctx, u.cs.Config.Images.Sync, hash)
}
