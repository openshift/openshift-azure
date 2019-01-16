package openshift

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	fakerp "github.com/openshift/openshift-azure/pkg/fakerp/shared"
	azuretls "github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func login(username string) (*api.Config, error) {
	dataDir, err := fakerp.FindDirectory(fakerp.DataDirectory)
	if err != nil {
		return nil, err
	}
	cs, err := managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
	if err != nil {
		return nil, err
	}

	var organization []string
	switch username {
	case "customer-cluster-admin":
		organization = []string{"osa-customer-admins"}
	case "customer-cluster-reader":
		organization = []string{"osa-customer-readers"}
	case "enduser":
		organization = []string{"system:authenticated", "system:authenticated:oauth"}
	case "admin":
		var c api.Config
		err := latest.Scheme.Convert(cs.Config.AdminKubeconfig, &c, nil)
		if err != nil {
			return nil, err
		}
		return &c, nil
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

	kc, err := makeKubeConfig(key, cert, cs.Config.Certificates.Ca.Cert, cs.Properties.FQDN, username, "default")
	var c api.Config
	err = latest.Scheme.Convert(kc, &c, nil)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func makeKubeConfig(clientKey *rsa.PrivateKey, clientCert, caCert *x509.Certificate, endpoint, username, namespace string) (*api.Config, error) {
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

	return &api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*api.Cluster{
			clustername: {
				Server:                   "https://" + endpoint,
				CertificateAuthorityData: caCertBytes,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			authinfoname: {
				ClientCertificateData: clientCertBytes,
				ClientKeyData:         clientKeyBytes,
			},
		},
		Contexts: map[string]*api.Context{
			contextname: {
				Cluster:   clustername,
				Namespace: namespace,
				AuthInfo:  authinfoname,
			},
		},
		CurrentContext: contextname,
	}, nil
}
