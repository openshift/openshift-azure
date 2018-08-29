package initialize

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/openshift/openshift-azure/pkg/api"
)

type azureStorageClient struct {
	accounts storage.AccountsClient
	storage  azstorage.Client
}

func newAzureClients(ctx context.Context, cs *api.OpenShiftManagedCluster) (*azureStorageClient, error) {
	config := auth.NewClientCredentialsConfig(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, err
	}

	clients := &azureStorageClient{}
	clients.accounts = storage.NewAccountsClient(cs.Properties.AzProfile.SubscriptionID)
	clients.accounts.Authorizer = authorizer

	keys, err := clients.accounts.ListKeys(context.Background(), cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
	if err != nil {
		return nil, err
	}

	clients.storage, err = azstorage.NewClient(cs.Config.ConfigStorageAccount, *(*keys.Keys)[0].Value, azstorage.DefaultBaseURL, azstorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	return clients, nil
}
