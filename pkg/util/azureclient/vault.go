package azureclient

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type VaultMgmtClient interface {
	VaultMgmtClientAddons
}

type vaultsMgmtClient struct {
	mgmtkeyvault.VaultsClient
}

var _ VaultMgmtClient = &vaultsMgmtClient{}

// NewVaultMgmtClient get a new client for management actions
func NewVaultMgmtClient(cfg auth.ClientCredentialsConfig, subscriptionID string) (VaultMgmtClient, error) {
	var err error
	var vmc vaultsMgmtClient
	vmc.VaultsClient = mgmtkeyvault.NewVaultsClient(subscriptionID)
	vmc.VaultsClient.Authorizer, err = cfg.Authorizer()
	return &vmc, err
}

type KeyVaultClient interface {
	KeyVaultClientAddons
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
}

type keyVaultsClient struct {
	keyvault.BaseClient
	vaultURL string
}

var _ KeyVaultClient = &keyVaultsClient{}

// NewKeyVaultClient get a new client for accessing vault values
func NewKeyVaultClient(cfg auth.ClientCredentialsConfig, vaultURL string) (KeyVaultClient, error) {
	var err error
	var kvc keyVaultsClient
	kvc.BaseClient = keyvault.New()
	kvc.vaultURL = vaultURL
	cfg.Resource = strings.TrimSuffix(azure.PublicCloud.KeyVaultEndpoint, "/") // beware of the leopard
	kvc.Authorizer, err = cfg.Authorizer()
	return &kvc, err
}
