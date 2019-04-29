package storage

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
)

type StorageRP struct {
	Log   *logrus.Entry
	Accts []storage.Account
	Blobs map[string]map[string][]byte
}

// FakeAccountsClient is a mock of AccountsClient interface
type FakeAccountsClient struct {
	rp *StorageRP
}

// NewFakeAccountsClient creates a new mock instance
func NewFakeAccountsClient(rp *StorageRP) *FakeAccountsClient {
	return &FakeAccountsClient{rp: rp}
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
	a.rp.Accts = append(a.rp.Accts, acct)
	return nil
}

// ListByResourceGroup mocks base method
func (a *FakeAccountsClient) ListByResourceGroup(context context.Context, resourceGroup string) (storage.AccountListResult, error) {
	return storage.AccountListResult{Value: &a.rp.Accts}, nil
}

// ListKeys mocks base method
func (a *FakeAccountsClient) ListKeys(context context.Context, resourceGroup, accountName string) (storage.AccountListKeysResult, error) {
	return storage.AccountListKeysResult{}, fmt.Errorf("FakeAccountsClient.ListKeys() not implemented")
}
