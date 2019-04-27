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

func (u *simpleUpgrader) initializeStorageClients(ctx context.Context) error {
	if u.storageClient == nil {
		if u.cs.Config.ConfigStorageAccountKey == "" {
			keys, err := u.accountsClient.ListKeys(ctx, u.cs.Properties.AzProfile.ResourceGroup, u.cs.Config.ConfigStorageAccount)
			if err != nil {
				return err
			}
			u.cs.Config.ConfigStorageAccountKey = *(*keys.Keys)[0].Value
		}

		var err error
		u.storageClient, err = storage.NewClient(u.log, u.cs.Config.ConfigStorageAccount, u.cs.Config.ConfigStorageAccountKey, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
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

func (u *simpleUpgrader) WriteStartupBlobs() error {
	u.log.Info("writing startup blobs")
	err := u.writeBlob(MasterStartupBlobName, u.cs)
	if err != nil {
		return err
	}

	workerCS := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			WorkerServicePrincipalProfile: api.ServicePrincipalProfile{
				ClientID: u.cs.Properties.WorkerServicePrincipalProfile.ClientID,
				Secret:   u.cs.Properties.WorkerServicePrincipalProfile.Secret,
			},
			AzProfile: api.AzProfile{
				TenantID:       u.cs.Properties.AzProfile.TenantID,
				SubscriptionID: u.cs.Properties.AzProfile.SubscriptionID,
				ResourceGroup:  u.cs.Properties.AzProfile.ResourceGroup,
			},
		},
		Location: u.cs.Location,
		Config: api.Config{
			PluginVersion: u.cs.Config.PluginVersion,
			ComponentLogLevel: api.ComponentLogLevel{
				Node: u.cs.Config.ComponentLogLevel.Node,
			},
			Certificates: api.CertificateConfig{
				Ca: api.CertKeyPair{
					Cert: u.cs.Config.Certificates.Ca.Cert,
				},
				NodeBootstrap: u.cs.Config.Certificates.NodeBootstrap,
			},
			Images: api.ImageConfig{
				Format:          u.cs.Config.Images.Format,
				Node:            u.cs.Config.Images.Node,
				ImagePullSecret: u.cs.Config.Images.ImagePullSecret,
			},
			NodeBootstrapKubeconfig: u.cs.Config.NodeBootstrapKubeconfig,
			SDNKubeconfig:           u.cs.Config.SDNKubeconfig,
		},
	}
	for _, app := range u.cs.Properties.AgentPoolProfiles {
		workerCS.Properties.AgentPoolProfiles = append(workerCS.Properties.AgentPoolProfiles, api.AgentPoolProfile{
			Role:   app.Role,
			VMSize: app.VMSize,
		})
	}

	return u.writeBlob(WorkerStartupBlobName, workerCS)
}

func (u *simpleUpgrader) CreateOrUpdateConfigStorageAccount(ctx context.Context) error {
	u.log.Info("creating/updating storage account")

	err := u.accountsClient.Create(ctx, u.cs.Properties.AzProfile.ResourceGroup, u.cs.Config.ConfigStorageAccount, azstorage.AccountCreateParameters{
		Sku: &azstorage.Sku{
			Name: azstorage.StandardLRS,
		},
		Kind:     azstorage.Storage,
		Location: &u.cs.Location,
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

func (u *simpleUpgrader) InitializeUpdateBlob(suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range u.cs.Properties.AgentPoolProfiles {
		h, err := u.hasher.HashScaleSet(u.cs, &app)
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
	return u.updateBlobService.Write(blob)
}

func (u *simpleUpgrader) ResetUpdateBlob() error {
	blob := updateblob.NewUpdateBlob()
	return u.updateBlobService.Write(blob)
}
