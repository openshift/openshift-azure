package enrich

import (
	"context"
	"time"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/operationalinsights"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/vault"
	"github.com/openshift/openshift-azure/pkg/util/workspace"
)

func StorageAccountKeys(ctx context.Context, azs storage.AccountsClient, cs *api.OpenShiftManagedCluster) error {
	if cs.Config.RegistryStorageAccountKey == "" {
		key, err := azs.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.RegistryStorageAccount)
		if err != nil {
			return err
		}
		cs.Config.RegistryStorageAccountKey = *(*key.Keys)[0].Value
	}

	if cs.Config.ConfigStorageAccount == "" {
		key, err := azs.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
		if err != nil {
			return err
		}
		cs.Config.ConfigStorageAccountKey = *(*key.Keys)[0].Value
	}

	return nil
}

func SASURIs(storageClient storage.Client, cs *api.OpenShiftManagedCluster) (err error) {
	now := time.Now().Add(-time.Hour)

	bsc := storageClient.GetBlobService()
	c := bsc.GetContainerReference("config") // TODO: should be using consts, need to merge packages

	cs.Config.MasterStartupSASURI, err = c.GetBlobReference("master-startup").GetSASURI(azstorage.BlobSASOptions{
		BlobServiceSASPermissions: azstorage.BlobServiceSASPermissions{
			Read: true,
		},
		SASOptions: azstorage.SASOptions{
			APIVersion: "2015-04-05",
			Start:      now,
			Expiry:     now.AddDate(5, 0, 0),
			UseHTTPS:   true,
		},
	})
	if err != nil {
		return
	}

	cs.Config.WorkerStartupSASURI, err = c.GetBlobReference("worker-startup").GetSASURI(azstorage.BlobSASOptions{
		BlobServiceSASPermissions: azstorage.BlobServiceSASPermissions{
			Read: true,
		},
		SASOptions: azstorage.SASOptions{
			APIVersion: "2015-04-05",
			Start:      now,
			Expiry:     now.AddDate(5, 0, 0),
			UseHTTPS:   true,
		},
	})
	return
}

func CertificatesFromVault(ctx context.Context, kvc keyvault.KeyVaultClient, cs *api.OpenShiftManagedCluster) error {
	kp, err := vault.GetSecret(ctx, kvc, cs.Properties.APICertProfile.KeyVaultSecretURL)
	if err != nil {
		return err
	}
	cs.Config.Certificates.OpenShiftConsole = *kp

	kp, err = vault.GetSecret(ctx, kvc, cs.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL)
	if err != nil {
		return err
	}
	cs.Config.Certificates.Router = *kp

	return nil
}

func MonitorIDAndKey(ctx context.Context, client operationalinsights.WorkspacesClient, cs *api.OpenShiftManagedCluster) error {
	var wID, wKey string
	var err error
	if cs.Properties.MonitorProfile.WorkspaceResourceID != "" {
		wID, wKey, err = workspace.GetWorkspaceInfo(ctx, client, cs.Properties.MonitorProfile.WorkspaceResourceID)
		if err != nil {
			return err
		}
	}
	cs.Properties.MonitorProfile.WorkspaceID = wID
	cs.Properties.MonitorProfile.WorkspaceKey = wKey
	return nil
}
