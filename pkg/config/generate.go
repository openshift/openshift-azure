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

	c.ImageConfigFormat = "openshift/origin-${component}:${version}"

	c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"
	c.MasterAPIImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.MasterControllersImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.BootstrapAutoapproverImage = "docker.io/openshift/origin-node:v3.10.0"
	c.ServiceCatalogImage = "quay.io/kargakis/servicecatalog:kubeconfig" // TODO: "docker.io/openshift/origin-service-catalog:v3.10.0"
	c.ProxyImage = "docker.io/jimminter/proxy:latest"
	c.SyncImage = "docker.io/jimminter/sync:latest"

	// TODO: need to cross-check all the below with acs-engine, especially SANs and IPs

	// Generate CAs
	cas := []struct {
		cn   string
		key  **rsa.PrivateKey
		cert **x509.Certificate
	}{
		{
			cn:   "etcd-signer",
			key:  &c.EtcdCaKey,
			cert: &c.EtcdCaCert,
		},
		{
			cn:   "openshift-signer",
			key:  &c.CaKey,
			cert: &c.CaCert,
		},
		// currently skipping the other frontproxy, doesn't seem to hurt
		{
			cn:   "openshift-frontproxy-signer",
			key:  &c.FrontProxyCaKey,
			cert: &c.FrontProxyCaCert,
		},
		{
			cn:   "openshift-service-serving-signer",
			key:  &c.ServiceSigningCaKey,
			cert: &c.ServiceSigningCaCert,
		},
		{
			cn:   "service-catalog-signer",
			key:  &c.ServiceCatalogCaKey,
			cert: &c.ServiceCatalogCaCert,
		},
	}
	for _, ca := range cas {
		if *ca.key, *ca.cert, err = tls.NewCA(ca.cn); err != nil {
			return
		}
	}

	certs := []struct {
		cn           string
		organization []string
		dnsNames     []string
		ipAddresses  []net.IP
		extKeyUsage  []x509.ExtKeyUsage
		signingKey   *rsa.PrivateKey
		signingCert  *x509.Certificate
		key          **rsa.PrivateKey
		cert         **x509.Certificate
	}{
		// Generate etcd certs
		{
			cn:          "master-etcd",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			signingKey:  c.EtcdCaKey,
			signingCert: c.EtcdCaCert,
			key:         &c.EtcdServerKey,
			cert:        &c.EtcdServerCert,
		},
		{
			cn:          "etcd-peer",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			signingKey:  c.EtcdCaKey,
			signingCert: c.EtcdCaCert,
			key:         &c.EtcdPeerKey,
			cert:        &c.EtcdPeerCert,
		},
		{
			cn:          "etcd-client",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			signingKey:  c.EtcdCaKey,
			signingCert: c.EtcdCaCert,
			key:         &c.EtcdClientKey,
			cert:        &c.EtcdClientCert,
		},
		// Generate openshift master certs
		{
			cn:           "system:admin",
			organization: []string{"system:cluster-admins", "system:masters"},
			extKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:          &c.AdminKey,
			cert:         &c.AdminCert,
		},
		{
			cn:          "aggregator-front-proxy",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			signingKey:  c.FrontProxyCaKey,
			signingCert: c.FrontProxyCaCert,
			key:         &c.AggregatorFrontProxyKey,
			cert:        &c.AggregatorFrontProxyCert,
		},
		// currently skipping etcd.server, doesn't seem to hurt
		{
			cn:           "system:openshift-node-admin",
			organization: []string{"system:node-admins"},
			extKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:          &c.MasterKubeletClientKey,
			cert:         &c.MasterKubeletClientCert,
		},
		{
			cn:          "system:master-proxy",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.MasterProxyClientKey,
			cert:        &c.MasterProxyClientCert,
		},
		{
			cn: "master-api",
			dnsNames: []string{
				"master-api",
				m.PublicHostname,
				"kubernetes",
				"kubernetes.default",
				"kubernetes.default.svc",
				"kubernetes.default.svc.cluster.local",
				"openshift",
				"openshift.default",
				"openshift.default.svc",
				"openshift.default.svc.cluster.local",
			},
			ipAddresses: []net.IP{net.ParseIP("172.30.0.1")},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.MasterServerKey,
			cert:        &c.MasterServerCert,
		},
		// currently skipping openshift-aggregator, doesn't seem to hurt
		{
			cn:           "system:openshift-master",
			organization: []string{"system:cluster-admins", "system:masters"},
			extKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:          &c.OpenShiftMasterKey,
			cert:         &c.OpenShiftMasterCert,
		},
		{
			cn:          "servicecatalog-api",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			signingKey:  c.ServiceCatalogCaKey,
			signingCert: c.ServiceCatalogCaCert,
			key:         &c.ServiceCatalogServerKey,
			cert:        &c.ServiceCatalogServerCert,
		},
	}
	for _, cert := range certs {
		if cert.signingKey == nil && cert.signingCert == nil {
			cert.signingKey, cert.signingCert = c.CaKey, c.CaCert
		}
		if *cert.key, *cert.cert, err = tls.NewCert(cert.cn, cert.organization, cert.dnsNames, cert.ipAddresses, cert.extKeyUsage, cert.signingKey, cert.signingCert); err != nil {
			return
		}
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
	if c.AdminKubeconfig, err = makeKubeConfig(c.AdminKey, c.AdminCert, c.CaCert, m.PublicHostname, "system:admin", "default"); err != nil {
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
	if c.NodeBootstrapKubeconfig, err = makeKubeConfig(c.NodeBootstrapKey, c.NodeBootstrapCert, c.CaCert, m.PublicHostname, "system:serviceaccount:openshift-infra:node-bootstrapper", "default"); err != nil {
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
