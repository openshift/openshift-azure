package fakerp

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"os"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/aadapp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/vault"
)

const (
	vaultKeyNamePublicHostname = "PublicHostname"
	vaultKeyNameRouter         = "Router"
)

func vaultName(rg string) string {
	return rg + "-vault"
}

func vaultURL(rg string) string {
	return fmt.Sprintf("https://%s.vault.azure.net", vaultName(rg))
}

type vaultManager struct {
	vc  azureclient.VaultMgmtClient
	spc azureclient.ServicePrincipalsClient
	kvc azureclient.KeyVaultClient
}

func newVaultManager(ctx context.Context, subscriptionID string) (*vaultManager, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	vaultauthorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyVaultClientAuthorizer)
	if err != nil {
		return nil, err
	}

	return &vaultManager{
		vc:  azureclient.NewVaultMgmtClient(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
		spc: azureclient.NewServicePrincipalsClient(ctx, os.Getenv("AZURE_TENANT_ID"), graphauthorizer),
		kvc: azureclient.NewKeyVaultClient(ctx, vaultauthorizer),
	}, nil
}

func (vm *vaultManager) writeTLSCertsToVault(ctx context.Context, cs *api.OpenShiftManagedCluster, vaultURL string) error {
	certs := []struct {
		vaultKeyName string
		params       tls.CertParams
	}{
		{
			vaultKeyName: vaultKeyNameRouter,
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
		{
			vaultKeyName: vaultKeyNamePublicHostname,
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: cs.Properties.PublicHostname,
				},
				DNSNames: []string{
					cs.Properties.PublicHostname,
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
		},
	}
	for _, cert := range certs {
		_, err := vm.kvc.GetSecret(ctx, vaultURL, cert.vaultKeyName, "")
		if err == nil {
			// let's assume it doesn't need updating
			continue
		}

		newkey, newcert, err := tls.NewCert(&cert.params)
		if err != nil {
			return err
		}
		err = vault.ImportCertificate(ctx, vm.kvc, vaultURL, cert.vaultKeyName, newkey, newcert)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vm *vaultManager) createOrUpdateVault(ctx context.Context, appID, tenantID, resourceGroup, location, vaultName string) error {
	tID, err := uuid.FromString(tenantID)
	if err != nil {
		return err
	}

	spObjID, err := aadapp.GetServicePrincipalObjectIDFromAppID(ctx, vm.spc, appID)
	if err != nil {
		return err
	}

	_, err = vm.vc.CreateOrUpdate(ctx, resourceGroup, vaultName, mgmtkeyvault.VaultCreateOrUpdateParameters{
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
					ObjectID: &spObjID,
					Permissions: &mgmtkeyvault.Permissions{
						Certificates: &[]mgmtkeyvault.CertificatePermissions{
							mgmtkeyvault.Import,
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

func (vm *vaultManager) deleteVault(ctx context.Context, subscriptionID, resourceGroup, vaultName string) error {
	_, err := vm.vc.Delete(ctx, resourceGroup, vaultName)
	return err
}
