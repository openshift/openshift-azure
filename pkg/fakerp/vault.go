package fakerp

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"os"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

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
	fakerpCAKey, fakerpCACert, err := tls.NewCA("external-signer")
	if err != nil {
		return err
	}

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
				SigningCert: fakerpCACert,
				SigningKey:  fakerpCAKey,
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
				SigningCert: fakerpCACert,
				SigningKey:  fakerpCAKey,
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

		// This chain should follow Certificate chain practices, where order is:
		// End-User Certificate
		// Intermediate Certificate
		// Root Certificate
		chain := api.CertKeyPairChain{
			Key: newkey,
			Certs: []*x509.Certificate{
				newcert, fakerpCACert,
			},
		}

		err = vault.ImportCertificate(ctx, vm.kvc, vaultURL, cert.vaultKeyName, chain)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vm *vaultManager) createOrUpdateVault(ctx context.Context, fakerpAppID, masterAppID, tenantID, resourceGroup, location, vaultName string) error {
	tID, err := uuid.FromString(tenantID)
	if err != nil {
		return err
	}

	fakerpObjID, err := aadapp.GetServicePrincipalObjectIDFromAppID(ctx, vm.spc, fakerpAppID)
	if err != nil {
		return err
	}

	masterObjID, err := aadapp.GetServicePrincipalObjectIDFromAppID(ctx, vm.spc, masterAppID)
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
					ObjectID: &fakerpObjID,
					Permissions: &mgmtkeyvault.Permissions{
						Certificates: &[]mgmtkeyvault.CertificatePermissions{
							mgmtkeyvault.Import,
						},
						Secrets: &[]mgmtkeyvault.SecretPermissions{
							mgmtkeyvault.SecretPermissionsGet,
						},
					},
				},
				{
					TenantID: &tID,
					ObjectID: &masterObjID,
					Permissions: &mgmtkeyvault.Permissions{
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
