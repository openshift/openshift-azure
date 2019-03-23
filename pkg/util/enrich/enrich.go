package enrich

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func CSStorageAccountKeys(ctx context.Context, azs azureclient.AccountsClient, cs *api.OpenShiftManagedCluster) error {
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
