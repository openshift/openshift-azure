package cluster

import (
	"bytes"
	"context"
	"encoding/json"

	azstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

func (u *SimpleUpgrader) initializeStorageClients(ctx context.Context) error {
	if u.StorageClient == nil {
		if u.Cs.Config.ConfigStorageAccountKey == "" {
			keys, err := u.AccountsClient.ListKeys(ctx, u.Cs.Properties.AzProfile.ResourceGroup, u.Cs.Config.ConfigStorageAccount)
			if err != nil {
				return err
			}
			u.Cs.Config.ConfigStorageAccountKey = *(*keys.Keys)[0].Value
		}

		var err error
		u.StorageClient, err = storage.NewClient(u.log, u.Cs.Config.ConfigStorageAccount, u.Cs.Config.ConfigStorageAccountKey, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
		if err != nil {
			return err
		}

		bsc := u.StorageClient.GetBlobService()
		u.UpdateBlobService = updateblob.NewBlobService(bsc)
	}

	return nil
}

func (u *SimpleUpgrader) writeBlob(blobName string, cs *api.OpenShiftManagedCluster) error {
	bsc := u.StorageClient.GetBlobService()
	c := bsc.GetContainerReference(ConfigContainerName)
	b := c.GetBlobReference(blobName)

	json, err := json.Marshal(cs)
	if err != nil {
		return err
	}

	return b.CreateBlockBlobFromReader(bytes.NewReader(json), nil)
}

func (u *SimpleUpgrader) WriteStartupBlobs() error {
	u.Log.Info("writing startup blobs")
	err := u.writeBlob(MasterStartupBlobName, u.Cs)
	if err != nil {
		return err
	}

	workerCS := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			WorkerServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientID: u.Cs.Properties.WorkerServicePrincipalProfile.ClientID,
				Secret:   u.Cs.Properties.WorkerServicePrincipalProfile.Secret,
			},
			AzProfile: api.AzProfile{
				TenantID:       u.Cs.Properties.AzProfile.TenantID,
				SubscriptionID: u.Cs.Properties.AzProfile.SubscriptionID,
				ResourceGroup:  u.Cs.Properties.AzProfile.ResourceGroup,
			},
		},
		Location: u.Cs.Location,
		Config: api.Config{
			PluginVersion: u.Cs.Config.PluginVersion,
			ComponentLogLevel: api.ComponentLogLevel{
				Node: u.Cs.Config.ComponentLogLevel.Node,
			},
			Certificates: api.CertificateConfig{
				Ca: api.CertKeyPair{
					Cert: u.Cs.Config.Certificates.Ca.Cert,
				},
				NodeBootstrap: u.Cs.Config.Certificates.NodeBootstrap,
			},
			Images: api.ImageConfig{
				Format:          u.Cs.Config.Images.Format,
				Node:            u.Cs.Config.Images.Node,
				ImagePullSecret: u.Cs.Config.Images.ImagePullSecret,
			},
			NodeBootstrapKubeconfig: u.Cs.Config.NodeBootstrapKubeconfig,
			SDNKubeconfig:           u.Cs.Config.SDNKubeconfig,
		},
	}
	for _, app := range u.Cs.Properties.AgentPoolProfiles {
		workerCS.Properties.AgentPoolProfiles = append(workerCS.Properties.AgentPoolProfiles, api.AgentPoolProfile{
			Role:   app.Role,
			VMSize: app.VMSize,
		})
	}

	return u.writeBlob(WorkerStartupBlobName, workerCS)
}

func (u *SimpleUpgrader) CreateOrUpdateConfigStorageAccount(ctx context.Context) error {
	u.Log.Info("creating/updating storage account")

	err := u.AccountsClient.Create(ctx, u.Cs.Properties.AzProfile.ResourceGroup, u.Cs.Config.ConfigStorageAccount, azstorage.AccountCreateParameters{
		Sku: &azstorage.Sku{
			Name: azstorage.StandardLRS,
		},
		Kind:     azstorage.Storage,
		Location: &u.Cs.Location,
		Tags: map[string]*string{
			"type": to.StringPtr("config"),
		},
		AccountPropertiesCreateParameters: &azstorage.AccountPropertiesCreateParameters{
			EnableHTTPSTrafficOnly: to.BoolPtr(true),
		},
	})
	if err != nil {
		return err
	}

	err = u.initializeStorageClients(ctx)
	if err != nil {
		return err
	}

	bsc := u.StorageClient.GetBlobService()

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

func (u *SimpleUpgrader) InitializeUpdateBlob(suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range u.Cs.Properties.AgentPoolProfiles {
		h, err := u.Hasher.HashScaleSet(u.Cs, &app)
		if err != nil {
			return err
		}

		switch app.Role {
		case api.AgentPoolProfileRoleMaster:
			for i := int64(0); i < app.Count; i++ {
				hostname := names.GetHostname(&app, suffix, i)
				blob.HostnameHashes[hostname] = h
			}

		default:
			blob.ScalesetHashes[names.GetScalesetName(&app, suffix)] = h
		}
	}
	return u.UpdateBlobService.Write(blob)
}

func (u *SimpleUpgrader) ResetUpdateBlob() error {
	blob := updateblob.NewUpdateBlob()
	return u.UpdateBlobService.Write(blob)
}
