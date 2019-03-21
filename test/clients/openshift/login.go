package openshift

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"strings"

	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	azuretls "github.com/openshift/openshift-azure/pkg/util/tls"
)

func login(username string, cs *internalapi.OpenShiftManagedCluster) (*v1.Config, error) {
	var organization []string
	switch username {
	case "customer-cluster-admin":
		organization = []string{"osa-customer-admins"}
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

	return makeKubeConfig(key, cert, cs.Config.Certificates.Ca.Cert, cs.Properties.FQDN, username, "default")
}

func makeKubeConfig(clientKey *rsa.PrivateKey, clientCert, caCert *x509.Certificate, endpoint, username, namespace string) (*v1.Config, error) {
	clustername := strings.Replace(endpoint, ".", "-", -1)
	authinfoname := username + "/" + clustername
	contextname := namespace + "/" + clustername + "/" + username

	caCertBytes, err := azuretls.CertAsBytes(caCert)
	if err != nil {
		return nil, err
	}
	clientCertBytes, err := azuretls.CertAsBytes(clientCert)
	if err != nil {
		return nil, err
	}
	clientKeyBytes, err := azuretls.PrivateKeyAsBytes(clientKey)
	if err != nil {
		return nil, err
	}

	return &v1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []v1.NamedCluster{
			{
				Name: clustername,
				Cluster: v1.Cluster{
					Server:                   "https://" + endpoint,
					CertificateAuthorityData: caCertBytes,
				},
			},
		},
		AuthInfos: []v1.NamedAuthInfo{
			{
				Name: authinfoname,
				AuthInfo: v1.AuthInfo{
					ClientCertificateData: clientCertBytes,
					ClientKeyData:         clientKeyBytes,
				},
			},
		},
		Contexts: []v1.NamedContext{
			{
				Name: contextname,
				Context: v1.Context{
					Cluster:   clustername,
					Namespace: namespace,
					AuthInfo:  authinfoname,
				},
			},
		},
		CurrentContext: contextname,
	}, nil
}
