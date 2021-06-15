package cluster

import (
	"context"
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

	return u.Interface.EnsureSyncPod(ctx, u.Cs.Config.Images.Sync, hash)
}

// PreSecretRotation prepares cluster for secret rotation.
// We are removing sync pod so it will not change the cluster while we are updating.
// We are removing ValidatingWebhookConfiguration as i is preventing cluster to start
// after certificate rotations as it is not running/admitted and k8s api server
// is not able to create anything else. It creates deadlock within the cluster.
// Once sync pod is re-deployed after upgrade/rotation is done it will be re-created.
func (u *Upgrade) PreSecretRotation(ctx context.Context) error {
	u.Log.Infof("removing sync pod")
	err := u.Interface.RemoveSyncPod(ctx)
	if err != nil {
		return err
	}
	u.Log.Infof("removing ValidatingWebhookConfiguration")
	err = u.Interface.RemoveValidatingWebhookConfiguration(ctx)
	if err != nil {
		return err
	}
	return nil
}
