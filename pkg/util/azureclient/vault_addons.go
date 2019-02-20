package azureclient

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/url"
	"path"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/satori/go.uuid"
)

type VaultMgmtClientAddons interface {
	CreateVault(ctx context.Context, appObjectID, subscriptionID, tenantID, resourceGroup, location, vaultName string) (string, error)
	DeleteVault(ctx context.Context, subscriptionID, resourceGroup, vaultName string) error
}

type KeyVaultClientAddons interface {
	StoreCertAndKey(ctx context.Context, name string, newkey *rsa.PrivateKey, newcert *x509.Certificate) error
}

func GetURLCertNameFromFullURL(kvURL string) (string, string, error) {
	u, err := url.Parse(kvURL)
	if err != nil {
		return "", "", err
	}
	certName := path.Base(u.Path)
	u.Path = ""
	vaultURL := u.String()
	return vaultURL, certName, nil
}

func (k *keyVaultsClient) StoreCertAndKey(ctx context.Context, name string, newkey *rsa.PrivateKey, newcert *x509.Certificate) error {
	buf := &bytes.Buffer{}
	b, err := x509.MarshalPKCS8PrivateKey(newkey) // Must be PKCS#8 for Azure Key Vault.
	if err != nil {
		return err
	}

	err = pem.Encode(buf, &pem.Block{Type: "PRIVATE KEY", Bytes: b})
	if err != nil {
		return err
	}

	err = pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: newcert.Raw})
	if err != nil {
		return err
	}

	_, err = k.ImportCertificate(ctx, k.vaultURL, name, keyvault.CertificateImportParameters{
		Base64EncodedCertificate: to.StringPtr(buf.String()),
		CertificatePolicy: &keyvault.CertificatePolicy{
			ID: to.StringPtr(name),
			SecretProperties: &keyvault.SecretProperties{
				ContentType: to.StringPtr("application/x-pem-file"),
			},
		},
	})
	return err
}

// CreateVault creates a new vault and returns the vaultURL
func (m *vaultsMgmtClient) CreateVault(ctx context.Context, appObjectID, subscriptionID, tenantID, resourceGroup, location, vaultName string) (string, error) {
	tID, err := uuid.FromString(tenantID)
	if err != nil {
		return "", err
	}
	aplist := []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tID,
			ObjectID: &appObjectID,
			Permissions: &mgmtkeyvault.Permissions{
				Certificates: &[]mgmtkeyvault.CertificatePermissions{
					mgmtkeyvault.Import,
					mgmtkeyvault.Get,
				},
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
				},
			},
		},
	}

	vault, err := m.CreateOrUpdate(
		ctx,
		resourceGroup,
		vaultName,
		mgmtkeyvault.VaultCreateOrUpdateParameters{
			Location: to.StringPtr(location),
			Properties: &mgmtkeyvault.VaultProperties{
				TenantID: &tID,
				Sku: &mgmtkeyvault.Sku{
					Family: to.StringPtr("A"),
					Name:   mgmtkeyvault.Standard,
				},
				AccessPolicies: &aplist,
			},
		},
	)
	if err != nil {
		return "", err
	}
	return *vault.Properties.VaultURI, nil
}

// DeleteVault deletes a vault
func (m *vaultsMgmtClient) DeleteVault(ctx context.Context, subscriptionID, resourceGroup, vaultName string) error {
	resp, err := m.Delete(ctx, resourceGroup, vaultName)
	if resp.StatusCode == 204 || resp.StatusCode == 404 {
		return nil
	}
	return err
}
