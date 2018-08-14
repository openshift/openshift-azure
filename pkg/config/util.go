package config

import (
	"crypto/rand"
	"io"
	"math/big"
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/tls"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

func makeKubeConfig(clientKey *tls.PrivateKey, clientCert, caCert *tls.Certificate, endpoint, username, namespace string) (*v1.Config, error) {
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
					Server: "https://" + endpoint,
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

func makeHtPasswd(username, password string) ([]byte, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return append([]byte(username+":"), b...), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadAtLeast(rand.Reader, b, n); err != nil {
		return nil, err
	}
	return b, nil
}

func randomStorageAccountName() (string, error) {
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, 24)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}

func randomString(length int) (string, error) {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}

func selectDNSNames(cs *acsapi.OpenShiftManagedCluster) {

	// Prefix values used to set arm and router k8s service dns annotations
	cs.Config.RouterLBCNamePrefix = strings.Split(cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN, ".")[0]
	cs.Config.MasterLBCNamePrefix = strings.Split(cs.Properties.FQDN, ".")[0]

	// Set PublicHostname to FQDN values if not specified
	if cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname == "" {
		cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname = cs.Properties.FQDN
	}
	if cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain == "" {
		cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain = cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN
	}
}
