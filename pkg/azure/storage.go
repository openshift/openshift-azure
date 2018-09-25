package azure

import (
	"context"
	"errors"

	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
)

// AccountStorageClient is minimal interface for azure AccountStorageClient
type AccountStorageClient interface {
	// mirrored methods
	ListKeys(context context.Context, resourceGroup, accountName string) (storage.AccountListKeysResult, error)
	ListByResourceGroup(context context.Context, resourceGroup string) (storage.AccountListResult, error)
	// custom methods
	GetStorageAccount(ctx context.Context, resourceGroup, typeTag string) (map[string]string, error)
	GetStorageAccountKey(ctx context.Context, resourceGroup, accountName string) (string, error)
}

// azAccountStorageClient implements AccountStorageClient.
type azAccountStorageClient struct {
	aClient storage.AccountsClient
}

// StorageClient is minimal inferface for azure StorageClient
type StorageClient interface {
	// mirrored methods
	GetContainerReference(name string) *azstorage.Container
}

// azDeploymentClient implements DeploymentClient.
type azStorageClient struct {
	azsClient azstorage.Client
	bsClient  azstorage.BlobStorageClient
}

// NewAccountStorageClient return AccountStorageClient implementation
func NewAccountStorageClient(ctx context.Context, clientID, clientSecret, tenantID, subscriptionID string) (AccountStorageClient, error) {
	aClient := storage.NewAccountsClient(subscriptionID)
	config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, err
	}

	aClient.Authorizer = authorizer

	return &azAccountStorageClient{
		aClient: aClient,
	}, nil
}

// ListKeys returns all keys of the storage account
func (az azAccountStorageClient) ListKeys(ctx context.Context, resourceGroup, accountName string) (result storage.AccountListKeysResult, err error) {
	return az.aClient.ListKeys(ctx, resourceGroup, accountName)
}

// NewStorageClient return StorageClient implementation
func NewStorageClient(ctx context.Context, key, accountName string) (StorageClient, error) {

	// get az storage client
	azsClient, err := azstorage.NewClient(accountName, key, azstorage.DefaultBaseURL, azstorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	// get blob service client
	bsClient := azsClient.GetBlobService()

	return &azStorageClient{
		azsClient: azsClient,
		bsClient:  bsClient,
	}, nil
}

func (az azStorageClient) GetContainerReference(name string) *azstorage.Container {
	return az.bsClient.GetContainerReference(name)
}

func (az azAccountStorageClient) GetStorageAccount(ctx context.Context, resourceGroup, typeTag string) (map[string]string, error) {

	accts, err := az.aClient.ListByResourceGroup(context.Background(), resourceGroup)
	if err != nil {
		return nil, err
	}

	var acct storage.Account
	var found bool
	for _, acct = range *accts.Value {
		found = acct.Tags["type"] != nil && *acct.Tags["type"] == typeTag
		if found {
			break
		}
	}
	if !found {
		return nil, errors.New("storage account not found")
	}

	keys, err := az.aClient.ListKeys(context.Background(), resourceGroup, *acct.Name)
	if err != nil {
		return nil, err
	}

	// TODO: Allow choosing between the two storage account keys to
	//// enable more convenient key rotation.
	result := map[string]string{
		"name": *acct.Name,
		"key":  *(*keys.Keys)[0].Value,
	}
	return result, nil
}

func (az azAccountStorageClient) GetStorageAccountKey(ctx context.Context, resourceGroup, accountName string) (string, error) {

	response, err := az.aClient.ListKeys(context.Background(), resourceGroup, accountName)
	if err != nil {
		return "", err
	}
	// TODO: Allow choosing between the two storage account keys to
	// enable more convenient key rotation.
	return *(((*response.Keys)[0]).Value), nil
}

func (az azAccountStorageClient) ListByResourceGroup(ctx context.Context, resourceGroup string) (storage.AccountListResult, error) {
	return az.aClient.ListByResourceGroup(ctx, resourceGroup)
}
