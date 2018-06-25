package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"io"
	"math/big"
	"net"
	"strings"

	"github.com/satori/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/tls"
)

func Generate(m *api.Manifest) (c *Config, err error) {
	c = &Config{}

	c.RoutingConfigSubdomain = "example.com"
	c.PublicHostname = "master-api-demo.104.45.157.35.nip.io"
	c.ImageConfigFormat = "openshift/origin-${component}:${version}"

	c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"
	c.MasterAPIImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.MasterControllersImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.BootstrapAutoapproverImage = "docker.io/openshift/origin-node:v3.10.0"
	c.ServiceCatalogImage = "docker.io/openshift/origin-service-catalog:v3.10.0"
	c.SyncImage = "docker.io/jimminter/sync:latest"

	// TODO: need to cross-check all the below with acs-engine, especially SANs and IPs

	// Generate CAs
	if c.EtcdCaKey, c.EtcdCaCert, err = tls.NewCA("etcd-signer"); err != nil {
		return
	}
	if c.CaKey, c.CaCert, err = tls.NewCA("openshift-signer"); err != nil {
		return
	}
	// currently skipping the other frontproxy, doesn't seem to hurt
	if c.FrontProxyCaKey, c.FrontProxyCaCert, err = tls.NewCA("openshift-frontproxy-signer"); err != nil {
		return
	}
	if c.ServiceSigningCaKey, c.ServiceSigningCaCert, err = tls.NewCA("openshift-service-serving-signer"); err != nil {
		return
	}
	if c.ServiceCatalogCaKey, c.ServiceCatalogCaCert, err = tls.NewCA("service-catalog-signer"); err != nil {
		return
	}

	// Generate etcd certs
	if c.EtcdServerKey, c.EtcdServerCert, err = tls.NewCert("master-etcd", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}
	if c.EtcdPeerKey, c.EtcdPeerCert, err = tls.NewCert("etcd-peer", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}
	if c.EtcdClientKey, c.EtcdClientCert, err = tls.NewCert("etcd-client", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}

	// Generate openshift master certs
	if c.AdminKey, c.AdminCert, err = tls.NewCert("system:admin", []string{"system:cluster-admins", "system:masters"}, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	if c.AggregatorFrontProxyKey, c.AggregatorFrontProxyCert, err = tls.NewCert("aggregator-front-proxy", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.FrontProxyCaKey, c.FrontProxyCaCert); err != nil {
		return
	}
	// currently skipping etcd.server, doesn't seem to hurt
	if c.MasterKubeletClientKey, c.MasterKubeletClientCert, err = tls.NewCert("system:openshift-node-admin", []string{"system:node-admins"}, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	if c.MasterProxyClientKey, c.MasterProxyClientCert, err = tls.NewCert("system:master-proxy", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	if c.MasterServerKey, c.MasterServerCert, err = tls.NewCert("master-api", nil, []string{"master-api", c.PublicHostname}, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	// currently skipping openshift-aggregator, doesn't seem to hurt
	if c.OpenShiftMasterKey, c.OpenShiftMasterCert, err = tls.NewCert("system:openshift-master", []string{"system:cluster-admins", "system:masters"}, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}

	if c.ServiceCatalogServerKey, c.ServiceCatalogServerCert, err = tls.NewCert("servicecatalog-api", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.ServiceCatalogCaKey, c.ServiceCatalogCaCert); err != nil {
		return
	}

	if c.ServiceAccountPrivateKey, err = tls.NewPrivateKey(); err != nil {
		return
	}
	c.ServiceAccountPublicKey = &c.ServiceAccountPrivateKey.PublicKey

	if c.SessionSecretAuth, err = randomBytes(24); err != nil {
		return
	}
	if c.SessionSecretEnc, err = randomBytes(24); err != nil {
		return
	}

	if c.HtPasswd, err = makeHtPasswd("demo", "demo"); err != nil {
		return
	}

	if c.MasterKubeconfig, err = makeKubeConfig(c.OpenShiftMasterKey, c.OpenShiftMasterCert, c.CaCert, "master-api", "system:openshift-master", "default"); err != nil {
		return
	}
	if c.AdminKubeconfig, err = makeKubeConfig(c.AdminKey, c.AdminCert, c.CaCert, c.PublicHostname, "system:admin", "default"); err != nil {
		return
	}

	if c.SSHPrivateKey, err = tls.NewPrivateKey(); err != nil {
		return
	}
	if c.SSHPublicKey, err = ssh.NewPublicKey(&c.SSHPrivateKey.PublicKey); err != nil {
		return
	}
	if c.NodeBootstrapKey, c.NodeBootstrapCert, err = tls.NewCert("system:serviceaccount:openshift-infra:node-bootstrapper", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	if c.NodeBootstrapKubeconfig, err = makeKubeConfig(c.NodeBootstrapKey, c.NodeBootstrapCert, c.CaCert, c.PublicHostname, "system:serviceaccount:openshift-infra:node-bootstrapper", "default"); err != nil {
		return
	}

	// needed by import
	// TODO: these need to be filled out sanely, and need to fully migrate the
	// service catalog over from impexp to helm.
	c.RouterIP = net.ParseIP("0.0.0.0")
	c.EtcdHostname = "garbage"
	if c.RegistryStorageAccount, err = randomStorageAccountName(); err != nil {
		return
	}
	c.RegistryAccountKey = "garbage"
	c.RegistryServiceIP = net.ParseIP("172.30.190.177") // TODO: choose a particular IP address?
	if c.RegistryHTTPSecret, err = randomBytes(32); err != nil {
		return nil, err
	}
	if c.AlertManagerProxySessionSecret, err = randomBytes(32); err != nil {
		return nil, err
	}
	if c.AlertsProxySessionSecret, err = randomBytes(32); err != nil {
		return nil, err
	}
	if c.PrometheusProxySessionSecret, err = randomBytes(32); err != nil {
		return nil, err
	}
	if c.ServiceCatalogClusterID, err = uuid.NewV4(); err != nil {
		return nil, err
	}
	// TODO: is it possible for the registry to use
	// service.alpha.openshift.io/serving-cert-secret-name?
	// TODO: remove nip.io
	c.RegistryKey, c.RegistryCert, err =
		tls.NewCert(c.RegistryServiceIP.String(), nil,
			[]string{"docker-registry-default." + c.RegistryServiceIP.String() + ".nip.io",
				"docker-registry.default.svc",
				"docker-registry.default.svc.cluster.local",
			},
			[]net.IP{c.RegistryServiceIP},
			[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			c.CaKey,
			c.CaCert)
	if err != nil {
		return nil, err
	}
	// TODO: the router CN and SANs should be configurables.
	c.RouterKey, c.RouterCert, err =
		tls.NewCert("*."+c.RouterIP.String()+".nip.io", nil,
			[]string{"*." + c.RouterIP.String() + ".nip.io",
				c.RouterIP.String() + ".nip.io",
			},
			nil,
			[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			c.CaKey,
			c.CaCert)
	if err != nil {
		return nil, err
	}

	return c, nil
}

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
