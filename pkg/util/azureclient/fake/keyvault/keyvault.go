package keyvault

import (
	"context"
	"fmt"

	azkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
)

type VaultRP struct {
	Log     *logrus.Entry
	Calls   []string
	Secrets []azkeyvault.SecretBundle
}

// FakeKeyVaultClient is a Fake of KeyVaultClient interface
type FakeKeyVaultClient struct {
	rp *VaultRP
}

// NewFakeKeyVaultClient creates a new Fake instance
func NewFakeKeyVaultClient(rp *VaultRP) keyvault.KeyVaultClient {
	return &FakeKeyVaultClient{rp: rp}
}

// GetSecret Fakes base method
func (kv *FakeKeyVaultClient) GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result azkeyvault.SecretBundle, err error) {
	kv.rp.Calls = append(kv.rp.Calls, "KeyVaultClient:GetSecret:"+secretName)
	for _, s := range kv.rp.Secrets {
		if *s.ID == secretName {
			return s, nil
		}
	}
	return azkeyvault.SecretBundle{}, fmt.Errorf("secret %s/%s not found", vaultBaseURL, secretName)
}

// ImportCertificate Fakes base method
func (kv *FakeKeyVaultClient) ImportCertificate(arg0 context.Context, arg1, arg2 string, arg3 azkeyvault.CertificateImportParameters) (azkeyvault.CertificateBundle, error) {
	return azkeyvault.CertificateBundle{}, fmt.Errorf("fake not implemented")
}
