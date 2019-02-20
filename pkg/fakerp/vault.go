package fakerp

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func vaultName(rg string) string {
	return rg + "-vault"
}

// WriteTLSCertsToVault write the TLS certs and keys into the vault
func WriteTLSCertsToVault(ctx context.Context, kvc azureclient.KeyVaultClient, cs *api.OpenShiftManagedCluster, vaultURL string) error {
	publicHostname := cs.Properties.FQDN
	if cs.Properties.PublicHostname != "" {
		publicHostname = cs.Properties.PublicHostname
	}

	cs.Properties.APICertProfile.KeyVaultSecretURL = vaultURL + "/secrets/PublicHostname"
	cs.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL = vaultURL + "/secrets/Router"
	c := &cs.Config
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
			cert.params.SigningKey, cert.params.SigningCert = c.Certificates.Ca.Key, c.Certificates.Ca.Cert
		}
		newkey, newcert, err := tls.NewCert(&cert.params)
		if err != nil {
			return err
		}
		err = kvc.StoreCertAndKey(ctx, cert.vaultKeyName, newkey, newcert)
		if err != nil {
			return err
		}
	}
	return nil
}
