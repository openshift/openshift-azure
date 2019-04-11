package fake

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
)

// FakeAccountsClient is a mock of AccountsClient interface
type FakeAccountsClient struct {
	az *AzureCloud
}

// NewFakeAccountsClient creates a new mock instance
func NewFakeAccountsClient(az *AzureCloud) *FakeAccountsClient {
	return &FakeAccountsClient{az: az}
}

// Create mocks base method
func (a *FakeAccountsClient) Create(ctx context.Context, resourceGroupName string, accountName string, parameters storage.AccountCreateParameters) error {
	acct := storage.Account{
		Name: &accountName,
		Sku:  parameters.Sku,
		Kind: parameters.Kind,
		Tags: parameters.Tags,
		Type: to.StringPtr("Microsoft.Storage/storageAccounts"),
	}
	a.az.Accts = append(a.az.Accts, acct)
	return nil
}

// ListByResourceGroup mocks base method
func (a *FakeAccountsClient) ListByResourceGroup(context context.Context, resourceGroup string) (storage.AccountListResult, error) {
	return storage.AccountListResult{Value: &a.az.Accts}, nil
}

// ListKeys mocks base method
func (a *FakeAccountsClient) ListKeys(context context.Context, resourceGroup, accountName string) (storage.AccountListKeysResult, error) {
	return storage.AccountListKeysResult{}, fmt.Errorf("FakeAccountsClient.ListKeys() not implemented")
}
