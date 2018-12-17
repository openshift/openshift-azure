package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest"
)

// AccountsClient is a minimal interface for azure AccountsClient
type AccountsClient interface {
	ListKeys(context context.Context, resourceGroup, accountName string) (storage.AccountListKeysResult, error)
	ListByResourceGroup(context context.Context, resourceGroup string) (storage.AccountListResult, error)
}

type accountsClient struct {
	storage.AccountsClient
}

var _ AccountsClient = &accountsClient{}

// NewAccountsClient returns a new AccountsClient
func NewAccountsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) AccountsClient {
	client := storage.NewAccountsClient(subscriptionID)
	setupClient(&client.Client, authorizer, languages)

	return &accountsClient{
		AccountsClient: client,
	}
}
