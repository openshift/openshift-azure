package openshift

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"

	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/kubeconfig"
	azuretls "github.com/openshift/openshift-azure/pkg/util/tls"
)

func login(username string, cs *internalapi.OpenShiftManagedCluster) (*v1.Config, error) {
	var organization []string
	switch username {
	case "customer-cluster-admin":
		organization = []string{"osa-customer-admins", "system:authenticated", "system:authenticated:oauth"}
	case "enduser":
		organization = []string{"system:authenticated", "system:authenticated:oauth"}
	case "admin":
		return cs.Config.AdminKubeconfig, nil
	default:
		return nil, fmt.Errorf("unknown username %q", username)
	}

	params := azuretls.CertParams{
		Subject: pkix.Name{
			CommonName:   username,
			Organization: organization,
		},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		SigningCert: cs.Config.Certificates.Ca.Cert,
		SigningKey:  cs.Config.Certificates.Ca.Key,
	}
	key, cert, err := azuretls.NewCert(&params)
	if err != nil {
		return nil, err
	}

	return kubeconfig.Make(key, cert, cs.Config.Certificates.Ca.Cert, cs.Properties.FQDN, username, "default")
}
