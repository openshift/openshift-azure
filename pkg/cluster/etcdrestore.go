package cluster

import (
	"context"
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
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

func (u *simpleUpgrader) EtcdRestoreDeleteMasterScaleSet(ctx context.Context) *api.PluginError {
	// We may need/want to delete all the scalesets in the future
	err := u.ssc.Delete(ctx, u.cs.Properties.AzProfile.ResourceGroup, names.MasterScalesetName)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepScaleSetDelete}
	}
	return nil
}

func (u *simpleUpgrader) EtcdRestoreDeleteMasterScaleSetHashes(ctx context.Context) *api.PluginError {
	uBlob, err := u.updateBlobService.Read()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
	}
	// delete only the master entries from the blob in order to
	// avoid unnecessary infra and compute rotations.
	uBlob.HostnameHashes = updateblob.HostnameHashes{}

	err = u.updateBlobService.Write(uBlob)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
	}
	return nil
}
