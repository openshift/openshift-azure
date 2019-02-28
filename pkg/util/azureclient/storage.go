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
	AccountsClientAddons
	Client
	Delete(context context.Context, resourceGroup, accountName string) (autorest.Response, error)
	GetOrCreate(context context.Context, resourceGroup, accountName string, createParams storage.AccountCreateParameters) (*storage.Account, error)
}

func (a *accountsClient) Client() autorest.Client {
	return a.AccountsClient.Client
}

type accountsClient struct {
	storage.AccountsClient
}

var _ AccountsClient = &accountsClient{}

// NewAccountsClient returns a new AccountsClient
func NewAccountsClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) AccountsClient {
	client := storage.NewAccountsClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

	return &accountsClient{
		AccountsClient: client,
	}
}

// GetOrCreate will create the account if it doesn't exist, or return the account if already there
func (c *accountsClient) GetOrCreate(ctx context.Context, resourceGroup, accountName string, createParams storage.AccountCreateParameters) (*storage.Account, error) {
	var acct storage.Account
	future, outerErr := c.AccountsClient.Create(ctx, resourceGroup, accountName, createParams)
	if outerErr != nil {
		acct, innerErr := c.AccountsClient.GetProperties(ctx, resourceGroup, accountName)
		if innerErr != nil {
			return nil, innerErr
		}
		return &acct, nil
	}
	err := future.WaitForCompletionRef(ctx, c.AccountsClient.Client)
	if err != nil {
		return nil, err
	}
	acct, err = c.AccountsClient.GetProperties(ctx, resourceGroup, accountName)
	if err != nil {
		return nil, err
	}
	return &acct, nil
}
