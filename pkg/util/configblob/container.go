package configblob

import (
	"context"
	"errors"

	azstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
)

// GetService is a helper function called by code running outside of the plugin.
// It returns the blob storage interface to the storage account containing the
// config blob, etcd backups, etc.
func GetService(ctx context.Context, log *logrus.Entry, cpc *cloudprovider.Config) (storage.BlobStorageClient, error) {
	authorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, "")
	if err != nil {
		return nil, err
	}

	acctsCli := storage.NewAccountsClient(ctx, log, cpc.SubscriptionID, authorizer)

	accts, err := acctsCli.ListByResourceGroup(ctx, cpc.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var acct azstorage.Account
	var found bool
	for _, acct = range *accts.Value {
		found = acct.Tags["type"] != nil && *acct.Tags["type"] == "config"
		if found {
			break
		}
	}
	if !found {
		return nil, errors.New("storage account not found")
	}

	keys, err := acctsCli.ListKeys(ctx, cpc.ResourceGroup, *acct.Name)
	if err != nil {
		return nil, err
	}

	storageCli, err := storage.NewClient(log, *acct.Name, *(*keys.Keys)[0].Value, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	return storageCli.GetBlobService(), nil
}
