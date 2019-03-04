package cluster

import (
	"context"
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
)

func (u *simpleUpgrader) EtcdBlobExists(ctx context.Context, blobName string) error {
	bsc := u.storageClient.GetBlobService()
	etcdContainer := bsc.GetContainerReference(EtcdBackupContainerName)
	blob := etcdContainer.GetBlobReference(blobName)
	exists, err := blob.Exists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("Blob %s does not exist", blobName)
	}
	return nil
}

func (u *simpleUpgrader) EtcdRestoreDeleteMasterScaleSet(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	// We may need/want to delete all the scalesets in the future
	err := u.ssc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, config.MasterScalesetName)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepScaleSetDelete}
	}
	return nil
}

func (u *simpleUpgrader) EtcdRestoreDeleteMasterScaleSetHashes(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	uBlob, err := u.updateBlobService.Read()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
	}
	// delete only the master entries from the blob in order to
	// avoid unnecessary infra and compute rotations.
	uBlob.InstanceHashes = updateblob.InstanceHashes{}

	err = u.updateBlobService.Write(uBlob)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
	}
	return nil
}
