package cluster

import (
	"bytes"
	"context"
	"encoding/json"

	azstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

func (u *simpleUpgrader) initializeStorageClients(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	if u.storageClient == nil {
		if cs.Config.ConfigStorageAccountKey == "" {
			keys, err := u.accountsClient.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
			if err != nil {
				return err
			}
			cs.Config.ConfigStorageAccountKey = *(*keys.Keys)[0].Value
		}

		var err error
		u.storageClient, err = storage.NewClient(cs.Config.ConfigStorageAccount, cs.Config.ConfigStorageAccountKey, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
		if err != nil {
			return err
		}

		bsc := u.storageClient.GetBlobService()
		u.updateBlobService = updateblob.NewBlobService(bsc)
	}

	return nil
}

func (u *simpleUpgrader) writeBlob(blobName string, cs *api.OpenShiftManagedCluster) error {
	bsc := u.storageClient.GetBlobService()
	c := bsc.GetContainerReference(ConfigContainerName)
	b := c.GetBlobReference(blobName)

	json, err := json.Marshal(cs)
	if err != nil {
		return err
	}

	return b.CreateBlockBlobFromReader(bytes.NewReader(json), nil)
}

func (u *simpleUpgrader) WriteSyncBlob(cs *api.OpenShiftManagedCluster) error {
	return u.writeBlob(SyncBlobName, cs)
}

func (u *simpleUpgrader) WriteStartupBlobs(cs *api.OpenShiftManagedCluster) error {
	err := u.writeBlob(MasterStartupBlobName, cs)
	if err != nil {
		return err
	}

	workerCS := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			WorkerServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientID: cs.Properties.WorkerServicePrincipalProfile.ClientID,
				Secret:   cs.Properties.WorkerServicePrincipalProfile.Secret,
			},
			AzProfile: api.AzProfile{
				TenantID:       cs.Properties.AzProfile.TenantID,
				SubscriptionID: cs.Properties.AzProfile.SubscriptionID,
				ResourceGroup:  cs.Properties.AzProfile.ResourceGroup,
			},
		},
		Location: cs.Location,
		Config: api.Config{
			ComponentLogLevel: api.ComponentLogLevel{
				Node: cs.Config.ComponentLogLevel.Node,
			},
			Certificates: api.CertificateConfig{
				Ca: api.CertKeyPair{
					Cert: cs.Config.Certificates.Ca.Cert,
				},
				NodeBootstrap: cs.Config.Certificates.NodeBootstrap,
			},
			Images: api.ImageConfig{
				Format:          cs.Config.Images.Format,
				Node:            cs.Config.Images.Node,
				ImagePullSecret: cs.Config.Images.ImagePullSecret,
			},
			NodeBootstrapKubeconfig: cs.Config.NodeBootstrapKubeconfig,
			SDNKubeconfig:           cs.Config.SDNKubeconfig,
		},
	}
	for _, app := range cs.Properties.AgentPoolProfiles {
		workerCS.Properties.AgentPoolProfiles = append(workerCS.Properties.AgentPoolProfiles, api.AgentPoolProfile{
			Role:   app.Role,
			VMSize: app.VMSize,
		})
	}

	return u.writeBlob(WorkerStartupBlobName, workerCS)
}

func (u *simpleUpgrader) CreateOrUpdateConfigStorageAccount(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	err := u.accountsClient.Create(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount, azstorage.AccountCreateParameters{
		Sku: &azstorage.Sku{
			Name: azstorage.StandardLRS,
		},
		Kind:     azstorage.Storage,
		Location: &cs.Location,
		Tags: map[string]*string{
			"type": to.StringPtr("config"),
		},
	})
	if err != nil {
		return err
	}

	err = u.initializeStorageClients(ctx, cs)
	if err != nil {
		return err
	}

	bsc := u.storageClient.GetBlobService()

	// cluster config container
	c := bsc.GetContainerReference(ConfigContainerName)
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	// etcd data container
	c = bsc.GetContainerReference(EtcdBackupContainerName)
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	// update container
	c = bsc.GetContainerReference(updateblob.UpdateContainerName)
	_, err = c.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	return nil
}

func (u *simpleUpgrader) InitializeUpdateBlob(cs *api.OpenShiftManagedCluster, suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range cs.Properties.AgentPoolProfiles {
		switch app.Role {
		case api.AgentPoolProfileRoleMaster:
			for i := int64(0); i < app.Count; i++ {
				h, err := u.hasher.HashMasterScaleSet(cs, &app, i)
				if err != nil {
					return err
				}

				hostname := config.GetHostname(&app, suffix, i)
				blob.HostnameHashes[hostname] = h
			}

		default:
			h, err := u.hasher.HashWorkerScaleSet(cs, &app)
			if err != nil {
				return err
			}
			blob.ScalesetHashes[config.GetScalesetName(&app, suffix)] = h
		}
	}
	var err error
	blob.SyncPodHash, err = u.hasher.HashSyncPod(cs)
	if err != nil {
		return err
	}
	return u.updateBlobService.Write(blob)
}

func (u *simpleUpgrader) ResetUpdateBlob(cs *api.OpenShiftManagedCluster) error {
	blob := updateblob.NewUpdateBlob()
	return u.updateBlobService.Write(blob)
}
