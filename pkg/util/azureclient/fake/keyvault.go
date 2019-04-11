package fake

import (
	"context"
	"fmt"

	keyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// FakeKeyVaultClient is a Fake of KeyVaultClient interface
type FakeKeyVaultClient struct {
	az *AzureCloud
}

// NewFakeKeyVaultClient creates a new Fake instance
func NewFakeKeyVaultClient(az *AzureCloud) azureclient.KeyVaultClient {
	return &FakeKeyVaultClient{az: az}
}

// GetSecret Fakes base method
func (kv *FakeKeyVaultClient) GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error) {
	for _, s := range kv.az.Secrets {
		if *s.ID == secretName {
			return s, nil
		}
	}
	return keyvault.SecretBundle{}, fmt.Errorf("secret %s/%s not found", vaultBaseURL, secretName)
}

// ImportCertificate Fakes base method
func (kv *FakeKeyVaultClient) ImportCertificate(arg0 context.Context, arg1, arg2 string, arg3 keyvault.CertificateImportParameters) (keyvault.CertificateBundle, error) {
	return keyvault.CertificateBundle{}, fmt.Errorf("fake not implemented")
}
