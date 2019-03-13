package vault

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"net/url"
	"path"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

func EnrichCSFromVault(ctx context.Context, kvc azureclient.KeyVaultClient, cs *api.OpenShiftManagedCluster) error {
	kp, err := getSecret(ctx, kvc, cs.Properties.APICertProfile.KeyVaultSecretURL)
	if err != nil {
		return err
	}
	cs.Config.Certificates.OpenShiftConsole = *kp

	kp, err = getSecret(ctx, kvc, cs.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL)
	if err != nil {
		return err
	}
	cs.Config.Certificates.Router = *kp

	return nil
}

// SplitSecretURL parses a key vault secret URL, e.g.
// https://myvault.vault.azure.net/secrets/mysecret, and returns the root vault
// URL and secret name, e.g. https://myvault.vault.azure.net/ and mysecret.
func SplitSecretURL(kvURL string) (string, string, error) {
	u, err := url.Parse(kvURL)
	if err != nil {
		return "", "", err
	}

	secretName := path.Base(u.Path)
	u.Path = ""
	vaultURL := u.String()

	return vaultURL, secretName, nil
}

func getSecret(ctx context.Context, kvc azureclient.KeyVaultClient, secretURL string) (*api.CertKeyPairChain, error) {
	vaultURL, secretName, err := SplitSecretURL(secretURL)
	if err != nil {
		return nil, err
	}

	bundle, err := kvc.GetSecret(ctx, vaultURL, secretName, "")
	if err != nil {
		return nil, err
	}

	key, err := tls.ParsePrivateKey([]byte(*bundle.Value))
	if err != nil {
		return nil, err
	}

	certs, err := tls.ParseCertChain([]byte(*bundle.Value))
	if err != nil {
		return nil, err
	}

	return &api.CertKeyPairChain{Key: key, Certs: certs}, nil
}

func ImportCertificate(ctx context.Context, kvc azureclient.KeyVaultClient, vaultURL, name string, chain api.CertKeyPairChain) error {
	buf := &bytes.Buffer{}
	b, err := x509.MarshalPKCS8PrivateKey(chain.Key) // Must be PKCS#8 for Azure Key Vault.
	if err != nil {
		return err
	}
	// This chain should follow Certificate chain practices, where order is:
	// End-User Certificate
	// Intermediate Certificate
	// Root Certificate
	err = pem.Encode(buf, &pem.Block{Type: "PRIVATE KEY", Bytes: b})
	if err != nil {
		return err
	}
	for _, cert := range chain.Certs {
		err = pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		if err != nil {
			return err
		}
	}

	_, err = kvc.ImportCertificate(ctx, vaultURL, name, keyvault.CertificateImportParameters{
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
