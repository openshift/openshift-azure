package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
)

type VaultMgmtClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, vaultName string, parameters mgmtkeyvault.VaultCreateOrUpdateParameters) (result mgmtkeyvault.Vault, err error)
	Delete(ctx context.Context, resourceGroupName string, vaultName string) (result autorest.Response, err error)
}

type vaultsMgmtClient struct {
	mgmtkeyvault.VaultsClient
}

var _ VaultMgmtClient = &vaultsMgmtClient{}

// NewVaultMgmtClient get a new client for management actions
func NewVaultMgmtClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) VaultMgmtClient {
	client := mgmtkeyvault.NewVaultsClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

	return &vaultsMgmtClient{
		VaultsClient: client,
	}
}

type KeyVaultClient interface {
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	ImportCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error)
}

type keyVaultsClient struct {
	keyvault.BaseClient
}

var _ KeyVaultClient = &keyVaultsClient{}

// NewKeyVaultClient gets a new client for accessing vault values.  Important:
// the authorizer supplied must have its resource set to
// "https://vault.azure.net"
func NewKeyVaultClient(ctx context.Context, authorizer autorest.Authorizer) KeyVaultClient {
	client := keyvault.New()
	setupClient(ctx, &client.Client, authorizer)

	return &keyVaultsClient{
		BaseClient: client,
	}
}
