package cluster

import (
	"bytes"
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
)

// UpdateSyncPod updates the sync pod.
func (u *simpleUpgrader) UpdateSyncPod(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	u.log.Infof("updating sync pod")

	blob, err := u.updateBlobService.Read()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateSyncPodReadBlob}
	}

	desiredHash, err := u.hasher.HashSyncPod(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateSyncPodHashSyncPod}
	}

	if bytes.Equal(blob.SyncPodHash, desiredHash) {
		u.log.Infof("skipping sync pod since it's already updated")
		return nil
	}

	u.log.Infof("deleting sync pod")
	err = u.kubeclient.DeletePod(ctx, "kube-system", "sync-master-000000")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateSyncPodDeletePod}
	}

	err = u.kubeclient.WaitForReadySyncPod(ctx)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateSyncPodWaitForReady}
	}

	blob.SyncPodHash = desiredHash

	if err := u.updateBlobService.Write(blob); err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateSyncPodUpdateBlob}
	}

	return nil
}
