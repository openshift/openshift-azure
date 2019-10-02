package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api/features"
)

// CreateOrUpdateSyncPod creates or updates the sync pod.
func (u *Upgrade) CreateOrUpdateSyncPod(ctx context.Context) error {
	u.Log.Infof("updating sync pod")

	err := u.writeBlob(SyncBlobName, u.Cs)
	if err != nil {
		return err
	}

	hash, err := u.Hasher.HashSyncPod(u.Cs)
	if err != nil {
		return err
	}

	if features.PrivateLinkEnabled(u.Cs) {
		u.Interface.EnablePrivateEndpointRoundTripper(u.Cs)
	}

	return u.Interface.EnsureSyncPod(ctx, u.Cs.Config.Images.Sync, hash)
}
