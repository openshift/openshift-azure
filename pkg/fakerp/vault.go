package fakerp

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/vault"
)

func vaultName(rg string) string {
	return rg + "-vault"
}

func vaultURL(rg string) string {
	return fmt.Sprintf("https://%s.vault.azure.net", vaultName(rg))
}

func writeTLSCertsToVault(ctx context.Context, kvc azureclient.KeyVaultClient, cs *api.OpenShiftManagedCluster, vaultURL string) error {
	publicHostname := config.Derived.PublicHostname(cs)

	certs := []struct {
		vaultKeyName string
		params       tls.CertParams
	}{
		{
			vaultKeyName: "Router",
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: cs.Properties.RouterProfiles[0].PublicSubdomain,
				},
				DNSNames: []string{
					cs.Properties.RouterProfiles[0].PublicSubdomain,
					"*." + cs.Properties.RouterProfiles[0].PublicSubdomain,
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
		},
		// Do not attempt to make the OpenShift console certificate self-signed
		// if cs.Properties == cs.FQDN:
		// https://github.com/openshift/openshift-azure/issues/307
		{
			vaultKeyName: "PublicHostname",
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: publicHostname,
				},
				DNSNames: []string{
					publicHostname,
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
		},
	}
	for _, cert := range certs {
		if cert.params.SigningKey == nil && cert.params.SigningCert == nil {
			cert.params.SigningKey, cert.params.SigningCert = cs.Config.Certificates.Ca.Key, cs.Config.Certificates.Ca.Cert
		}
		newkey, newcert, err := tls.NewCert(&cert.params)
		if err != nil {
			return err
		}
		err = vault.ImportCertificate(ctx, kvc, vaultURL, cert.vaultKeyName, newkey, newcert)
		if err != nil {
			return err
		}
	}
	return nil
}

func createVault(ctx context.Context, vc azureclient.VaultMgmtClient, appObjectID, tenantID, resourceGroup, location, vaultName string) error {
	tID, err := uuid.FromString(tenantID)
	if err != nil {
		return err
	}

	_, err = vc.CreateOrUpdate(ctx, resourceGroup, vaultName, mgmtkeyvault.VaultCreateOrUpdateParameters{
		Location: to.StringPtr(location),
		Properties: &mgmtkeyvault.VaultProperties{
			TenantID: &tID,
			Sku: &mgmtkeyvault.Sku{
				Family: to.StringPtr("A"),
				Name:   mgmtkeyvault.Standard,
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					TenantID: &tID,
					ObjectID: &appObjectID,
					Permissions: &mgmtkeyvault.Permissions{
						Certificates: &[]mgmtkeyvault.CertificatePermissions{
							mgmtkeyvault.Import,
							// mgmtkeyvault.Get,
						},
						Secrets: &[]mgmtkeyvault.SecretPermissions{
							mgmtkeyvault.SecretPermissionsGet,
						},
					},
				},
			},
		},
	})
	return err
}

func deleteVault(ctx context.Context, m azureclient.VaultMgmtClient, subscriptionID, resourceGroup, vaultName string) error {
	_, err := m.Delete(ctx, resourceGroup, vaultName)
	return err
}
