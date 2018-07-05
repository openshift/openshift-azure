package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"io"
	"math/big"
	"net"
	"os"
	"strings"

	"github.com/satori/uuid"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/tls"
)

func Generate(m *api.Manifest, c *Config) (err error) {
	c.Version = versionLatest

	c.ImageOffer = "osa-preview"
	c.ImagePublisher = "redhat"
	c.ImageSKU = "origin_310"
	c.ImageVersion = "latest"

	c.ImageConfigFormat = "openshift/origin-${component}:${version}"

	c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"
	c.MasterAPIImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.MasterControllersImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.NodeImage = "docker.io/openshift/origin-node:v3.10.0"
	c.ServiceCatalogImage = "quay.io/kargakis/servicecatalog:kubeconfig" // TODO: "docker.io/openshift/origin-service-catalog:v3.10.0"
	c.TunnelImage = "docker.io/jimminter/tunnel:latest"
	c.SyncImage = "docker.io/jimminter/sync:latest"
	c.TemplateServiceBrokerImage = "docker.io/openshift/origin-template-service-broker:v3.10"
	c.PrometheusNodeExporterImage = "openshift/prometheus-node-exporter:v0.15.2"
	c.RegistryImage = "openshift/origin-docker-registry:v3.10.0"
	c.RouterImage = "openshift/origin-haproxy-router:v3.10.0"
	c.AzureCLIImage = "docker.io/microsoft/azure-cli:latest"
	c.RegistryConsoleImage = "cockpit/kubernetes:latest"
	c.AnsibleServiceBrokerImage = "ansibleplaybookbundle/origin-ansible-service-broker:latest"
	c.WebConsoleImage = "openshift/origin-web-console:v3.10.0"
	c.OAuthProxyImage = "openshift/oauth-proxy:v1.0.0"
	c.PrometheusImage = "openshift/prometheus:v2.2.1"
	c.PrometheusAlertBufferImage = "openshift/prometheus-alert-buffer:v0.0.2"
	c.PrometheusAlertManagerImage = "openshift/prometheus-alertmanager:v0.14.0"

	c.TunnelHostname = strings.Replace(m.PublicHostname, "openshift", "openshift-tunnel", 1)

	c.ImageResourceGroup = os.Getenv("IMAGE_RESOURCEGROUP")
	c.ImageResourceName = os.Getenv("IMAGE_RESOURCENAME")

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
		if *ca.key != nil && *ca.cert != nil {
			continue
		}
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
			cn: m.PublicHostname,
			dnsNames: []string{
				m.PublicHostname,
				"master-api",
				"kubernetes",
				"kubernetes.default",
				"kubernetes.default.svc",
				"kubernetes.default.svc.cluster.local",
			},
			ipAddresses: []net.IP{net.ParseIP("172.30.0.1")},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.MasterServerKey,
			cert:        &c.MasterServerCert,
		},
		{
			cn:          c.TunnelHostname,
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.TunnelKey,
			cert:        &c.TunnelCert,
		},
		{
			cn:           "system:openshift-master",
			organization: []string{"system:cluster-admins", "system:masters"},
			extKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:          &c.OpenShiftMasterKey,
			cert:         &c.OpenShiftMasterCert,
		},
		{
			cn: "servicecatalog-api",
			dnsNames: []string{
				"servicecatalog-api",
				"apiserver.kube-service-catalog.svc", // TODO: unclear how safe this is
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			signingKey:  c.ServiceCatalogCaKey,
			signingCert: c.ServiceCatalogCaCert,
			key:         &c.ServiceCatalogServerKey,
			cert:        &c.ServiceCatalogServerCert,
		},
		{
			cn:          "system:serviceaccount:kube-service-catalog:service-catalog-apiserver",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.ServiceCatalogAPIClientKey,
			cert:        &c.ServiceCatalogAPIClientCert,
		},
		{
			cn:          "system:serviceaccount:openshift-infra:bootstrap-autoapprover",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.BootstrapAutoapproverKey,
			cert:        &c.BootstrapAutoapproverCert,
		},
		{
			cn:          "system:serviceaccount:openshift-infra:node-bootstrapper",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.NodeBootstrapKey,
			cert:        &c.NodeBootstrapCert,
		},
		{
			cn: m.RoutingConfigSubdomain,
			dnsNames: []string{
				m.RoutingConfigSubdomain,
				"*." + m.RoutingConfigSubdomain,
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.RouterKey,
			cert:        &c.RouterCert,
		},
		{
			cn: "docker-registry-default." + m.RoutingConfigSubdomain,
			dnsNames: []string{
				"docker-registry-default." + m.RoutingConfigSubdomain,
				"docker-registry.default.svc",
				"docker-registry.default.svc.cluster.local",
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.RegistryKey,
			cert:        &c.RegistryCert,
		},
	}
	for _, cert := range certs {
		if cert.signingKey == nil && cert.signingCert == nil {
			cert.signingKey, cert.signingCert = c.CaKey, c.CaCert
		}
		if *cert.key != nil && *cert.cert != nil &&
			(*cert.cert).CheckSignatureFrom(cert.signingCert) == nil {
			continue
		}
		if *cert.key, *cert.cert, err = tls.NewCert(cert.cn, cert.organization, cert.dnsNames, cert.ipAddresses, cert.extKeyUsage, cert.signingKey, cert.signingCert); err != nil {
			return
		}
	}

	secrets := []struct {
		secret *[]byte
		n      int
	}{
		{
			secret: &c.SessionSecretAuth,
			n:      24,
		},
		{
			secret: &c.SessionSecretEnc,
			n:      24,
		},
		{
			secret: &c.RegistryHTTPSecret,
		},
		{
			secret: &c.AlertManagerProxySessionSecret,
		},
		{
			secret: &c.AlertsProxySessionSecret,
		},
		{
			secret: &c.PrometheusProxySessionSecret,
		},
	}
	for _, s := range secrets {
		if len(*s.secret) != 0 {
			continue
		}
		if s.n == 0 {
			s.n = 32
		}
		if *s.secret, err = randomBytes(s.n); err != nil {
			return
		}
	}

	kubeconfigs := []struct {
		clientKey  *rsa.PrivateKey
		clientCert *x509.Certificate
		endpoint   string
		username   string
		namespace  string
		kubeconfig **v1.Config
	}{
		{
			clientKey:  c.OpenShiftMasterKey,
			clientCert: c.OpenShiftMasterCert,
			endpoint:   "master-api",
			username:   "system:openshift-master",
			kubeconfig: &c.MasterKubeconfig,
		},
		{
			clientKey:  c.ServiceCatalogAPIClientKey,
			clientCert: c.ServiceCatalogAPIClientCert,
			endpoint:   "master-api",
			username:   "system:serviceaccount:kube-service-catalog:service-catalog-apiserver",
			namespace:  "kube-service-catalog",
			kubeconfig: &c.ServiceCatalogAPIKubeconfig,
		},
		{
			clientKey:  c.BootstrapAutoapproverKey,
			clientCert: c.BootstrapAutoapproverCert,
			endpoint:   "master-api",
			username:   "system:serviceaccount:openshift-infra:bootstrap-autoapprover",
			namespace:  "openshift-infra",
			kubeconfig: &c.BootstrapAutoapproverKubeconfig,
		},
		{
			clientKey:  c.AdminKey,
			clientCert: c.AdminCert,
			endpoint:   m.PublicHostname,
			username:   "system:admin",
			kubeconfig: &c.AdminKubeconfig,
		},
		{
			clientKey:  c.NodeBootstrapKey,
			clientCert: c.NodeBootstrapCert,
			endpoint:   m.PublicHostname,
			username:   "system:serviceaccount:openshift-infra:node-bootstrapper",
			kubeconfig: &c.NodeBootstrapKubeconfig,
		},
	}
	for _, kc := range kubeconfigs {
		if kc.namespace == "" {
			kc.namespace = "default"
		}
		if *kc.kubeconfig, err = makeKubeConfig(kc.clientKey, kc.clientCert, c.CaCert, kc.endpoint, kc.username, kc.namespace); err != nil {
			return
		}
	}

	if c.ServiceAccountKey == nil {
		if c.ServiceAccountKey, err = tls.NewPrivateKey(); err != nil {
			return
		}
	}

	if len(c.HtPasswd) == 0 {
		if c.HtPasswd, err = makeHtPasswd("demo", "demo"); err != nil {
			return
		}
	}

	if c.SSHKey == nil {
		if c.SSHKey, err = tls.NewPrivateKey(); err != nil {
			return
		}
	}

	if len(c.RegistryStorageAccount) == 0 {
		if c.RegistryStorageAccount, err = randomStorageAccountName(); err != nil {
			return
		}
	}

	if uuid.Equal(c.ServiceCatalogClusterID, uuid.Nil) {
		if c.ServiceCatalogClusterID, err = uuid.NewV4(); err != nil {
			return
		}
	}

	return
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
