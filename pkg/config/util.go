package config

import (
	"crypto/rsa"
	"crypto/x509"
	"strings"

	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/tls"
)

func makeKubeConfig(clientKey *rsa.PrivateKey, clientCert, caCert *x509.Certificate, endpoint, username, namespace string) (*v1.Config, error) {
	clustername := strings.Replace(endpoint, ".", "-", -1)
	authinfoname := username + "/" + clustername
	contextname := namespace + "/" + clustername + "/" + username

	caCertBytes, err := tls.CertAsBytes(caCert)
	if err != nil {
		return nil, err
	}
	clientCertBytes, err := tls.CertAsBytes(clientCert)
	if err != nil {
		return nil, err
	}
	clientKeyBytes, err := tls.PrivateKeyAsBytes(clientKey)
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
