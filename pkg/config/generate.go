package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"strings"

	acsapi "github.com/Azure/acs-engine/pkg/api"
	"github.com/satori/uuid"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/jim-minter/azure-helm/pkg/tls"
)

// ORIGIN Images
//		MasterEtcdImage:             "quay.io/coreos/etcd",
//		MasterAPIImage:              "docker.io/openshift/origin-control-plane",
//		MasterControllersImage:      "docker.io/openshift/origin-control-plane",
//		NodeImage:                   "docker.io/openshift/origin-node",
//		ServiceCatalogImage:         "docker.io/openshift/origin-service-catalog",
//		TemplateServiceBrokerImage:  "docker.io/openshift/origin-template-service-broker",
//		PrometheusNodeExporterImage: "docker.io/openshift/prometheus-node-exporter",
//		RegistryImage:               "docker.io/openshift/origin-docker-registry",
//		RouterImage:                 "docker.io/openshift/origin-haproxy-router",
//		RegistryConsoleImage:        "docker.io/cockpit/kubernetes",
//		AnsibleServiceBrokerImage:   "docker.io/ansibleplaybookbundle/origin-ansible-service-broker",
//		WebConsoleImage:             "docker.io/openshift/origin-web-console",
//		OAuthProxyImage:             "docker.io/openshift/oauth-proxy",
//		PrometheusImage:             "docker.io/openshift/prometheus",
//		PrometheusAlertBufferImage:  "docker.io/openshift/prometheus-alert-buffer",
//		PrometheusAlertManagerImage: "docker.io/openshift/prometheus-alertmanager",
//
//		TunnelImage:   "docker.io/jimminter/tunnel",
//		SyncImage:     "docker.io/jimminter/sync",
//		AzureCLIImage: "docker.io/microsoft/azure-cli",
//
// OSA Images
//		MasterAPIImage:              "registry.access.redhat.com/openshift3/ose-control-plane",
//		MasterControllersImage:      "registry.access.redhat.com/openshift3/ose-control-plane",
//		NodeImage:                   "registry.access.redhat.com/openshift3/ose-node",
//		ServiceCatalogImage:         "registry.access.redhat.com/openshift3/ose-service-catalog",
//		TemplateServiceBrokerImage:  "registry.access.redhat.com/openshif3t/ose-template-service-broker",
//		PrometheusNodeExporterImage: "registry.access.redhat.com/openshift3/prometheus-node-exporter",
//		RegistryImage:               "registry.access.redhat.com/openshift3/ose-docker-registry",
//		RouterImage:                 "registry.access.redhat.com/openshift3/ose-haproxy-router",
//		RegistryConsoleImage:        "registry.access.redhat.com/openshift3/registry-console",
//		AnsibleServiceBrokerImage:   "registry.access.redhat.com/openshift3/ose-ansible-service-broker",
//		WebConsoleImage:             "registry.access.redhat.com/openshift3/ose-web-console",
//		OAuthProxyImage:             "registry.access.redhat.com/openshift3/oauth-proxy",
//		PrometheusImage:             "registry.access.redhat.com/openshift3/prometheus",
//		PrometheusAlertBufferImage:  "registry.access.redhat.com/openshift3/prometheus-alert-buffer",
//		PrometheusAlertManagerImage: "registry.access.redhat.com/openshift3/prometheus-alertmanager",
//
//		TunnelImage:   "docker.io/jimminter/tunnel",
//		SyncImage:     "docker.io/jimminter/sync",
//		AzureCLIImage: "docker.io/microsoft/azure-cli

func selectNodeImage(cs *acsapi.ContainerService, c *Config) {
	c.ImagePublisher = "redhat"
	c.ImageOffer = "osa-preview"
	c.ImageVersion = "latest"

	switch os.Getenv("DEPLOY_OS") {
	case "":
		c.ImageSKU = "osa_" + strings.Replace(cs.Properties.OrchestratorProfile.OpenShiftConfig.OpenShiftVersion, ".", "", -1)
	case "centos7":
		c.ImageSKU = "origin_" + strings.Replace(cs.Properties.OrchestratorProfile.OpenShiftConfig.OpenShiftVersion, ".", "", -1)
	}

	c.ImageResourceGroup = os.Getenv("IMAGE_RESOURCEGROUP")
	c.ImageResourceName = os.Getenv("IMAGE_RESOURCENAME")
}

func selectContainerImagesOrigin(cs *acsapi.ContainerService, c *Config) {
	// TODO:
	// Publish tunnel, sync, images
	c.ImageConfigFormat = "openshift/origin-${component}:${version}"
	switch cs.Properties.OrchestratorProfile.OpenShiftConfig.OpenShiftVersion {
	case "3.11":
		c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"
		c.MasterAPIImage = "docker.io/openshift/origin-control-plane:v3.11"
		c.MasterControllersImage = "docker.io/openshift/origin-control-plane:v3.11"
		c.NodeImage = "docker.io/openshift/origin-node:v3.11"
		c.ServiceCatalogImage = "docker.io/openshift/origin-service-catalog:v3.11"
		c.TemplateServiceBrokerImage = "docker.io/openshift/origin-template-service-broker:v3.11"
		c.PrometheusNodeExporterImage = "docker.io/openshift/prometheus-node-exporter:v0.15.2"
		c.RegistryImage = "docker.io/openshift/origin-docker-registry:v3.11"
		c.RouterImage = "docker.io/openshift/origin-haproxy-router:v3.11"
		c.RegistryConsoleImage = "docker.io/cockpit/kubernetes:latest"
		c.AnsibleServiceBrokerImage = "docker.io/ansibleplaybookbundle/origin-ansible-service-broker:latest"
		c.WebConsoleImage = "docker.io/openshift/origin-web-console:v3.11"
		c.OAuthProxyImage = "docker.io/openshift/oauth-proxy:v3.11"
		c.PrometheusImage = "docker.io/openshift/prometheusv2.2.1"
		c.PrometheusAlertBufferImage = "docker.io/openshift/prometheus-alert-buffer:v0.0.2"
		c.PrometheusAlertManagerImage = "docker.io/openshift/prometheus-alertmanager:v0.14.0"

		c.TunnelImage = "docker.io/jimminter/tunnel:latest"
		c.SyncImage = "docker.io/jimminter/sync:latest"
		c.AzureCLIImage = "docker.io/microsoft/azure-cli:latest"

	case "3.10":
		c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"
		c.MasterAPIImage = "docker.io/openshift/origin-control-plane:v3.10"
		c.MasterControllersImage = "docker.io/openshift/origin-control-plane:v3.10"
		c.NodeImage = "docker.io/openshift/origin-node:v3.10"
		c.ServiceCatalogImage = "docker.io/openshift/origin-service-catalog:v3.10"
		c.TemplateServiceBrokerImage = "docker.io/openshift/origin-template-service-broker:v3.10"
		c.PrometheusNodeExporterImage = "docker.io/openshift/prometheus-node-exporter:v0.15.2"
		c.RegistryImage = "docker.io/openshift/origin-docker-registry:v3.10"
		c.RouterImage = "docker.io/openshift/origin-haproxy-router:v3.10"
		c.RegistryConsoleImage = "docker.io/cockpit/kubernetes:latest"
		c.AnsibleServiceBrokerImage = "docker.io/ansibleplaybookbundle/origin-ansible-service-broker:latest"
		c.WebConsoleImage = "docker.io/openshift/origin-web-console:v3.10"
		c.OAuthProxyImage = "docker.io/openshift/oauth-proxy:v3.10"
		c.PrometheusImage = "docker.io/openshift/prometheusv2.2.1"
		c.PrometheusAlertBufferImage = "docker.io/openshift/prometheus-alert-buffer:v0.0.2"
		c.PrometheusAlertManagerImage = "docker.io/openshift/prometheus-alertmanager:v0.14.0"

		c.TunnelImage = "docker.io/jimminter/tunnel:latest"
		c.SyncImage = "docker.io/jimminter/sync:latest"
		c.AzureCLIImage = "docker.io/microsoft/azure-cli:latest"
	}
}

func selectContainerImagesOSA(cs *acsapi.ContainerService, c *Config) {
	// TODO:
	// Publish tunnel, sync, images
	// After GA change ImageConfigFormat, default registry and update default tags for released version
	c.ImageConfigFormat = "openshift3/ose-${component}:${version}"
	switch cs.Properties.OrchestratorProfile.OpenShiftConfig.OpenShiftVersion {
	case "3.10":
		c.ImageConfigFormat = "registry.reg-aws.openshift.com/openshift3/ose-${component}:${version}"
		c.ContainerPullSecret = configureDevRegPullSecret("registry.reg-aws.openshift.com")

		c.MasterEtcdImage = "registry.reg-aws.openshift.com/rhel3/etcd:3.2.22"
		c.MasterAPIImage = "registry.reg-aws.openshift.com/openshift3/ose-control-plane:v3.10"
		c.MasterControllersImage = "registry.reg-aws.openshift.com/openshift3/ose-control-plane:v3.10"
		c.NodeImage = "registry.reg-aws.openshift.com/openshift3/ose-node:v3.10"
		c.ServiceCatalogImage = "registry.reg-aws.openshift.com/openshift3/ose-service-catalog:v3.10"
		c.TemplateServiceBrokerImage = "registry.reg-aws.openshift.com/openshift3/ose-template-service-broker:v3.10"
		c.PrometheusNodeExporterImage = "registry.reg-aws.openshift.com/openshift3/prometheus-node-exporter:v0.15.2"
		c.RegistryImage = "registry.reg-aws.openshift.com/openshift3/ose-docker-registry:v3.10"
		c.RouterImage = "registry.reg-aws.openshift.com/openshift3/ose-haproxy-router:v3.10"
		c.RegistryConsoleImage = "registry.reg-aws.openshift.com/openshift3/registry-console:v3.10"
		c.AnsibleServiceBrokerImage = "registry.reg-aws.openshift.com/openshift3/ose-ansible-service-broker:v3.10"
		c.WebConsoleImage = "registry.reg-aws.openshift.com/openshift3/origin-web-console:v3.10"
		c.OAuthProxyImage = "registry.reg-aws.openshift.com/openshift3/oauth-proxy:v3.10"
		c.PrometheusImage = "registry.reg-aws.openshift.com/openshift3/prometheus:v2.2.1"
		c.PrometheusAlertBufferImage = "registry.reg-aws.openshift.com/openshift3/prometheus-alert-buffer:v0.0.2"
		c.PrometheusAlertManagerImage = "registry.reg-aws.openshift.com/openshift3/prometheus-alertmanager:v0.14.0"

		c.TunnelImage = "docker.io/jimminter/tunnel:latest"
		c.SyncImage = "docker.io/jimminter/sync:latest"
		c.AzureCLIImage = "docker.io/microsoft/azure-cli:latest"

	case "3.11":
		c.ImageConfigFormat = "registry.reg-aws.openshift.com/openshift3/ose-${component}:${version}"
		c.ContainerPullSecret = configureDevRegPullSecret("registry.reg-aws.openshift.com")

		c.MasterEtcdImage = "registry.reg-aws.openshift.com/rhel3/etcd:3.2.22"
		c.MasterAPIImage = "registry.reg-aws.openshift.com/openshift3/ose-control-plane:v3.11"
		c.MasterControllersImage = "registry.reg-aws.openshift.com/openshift3/ose-control-plane:v3.11"
		c.NodeImage = "registry.reg-aws.openshift.com/openshift3/ose-node:v3.11"
		c.ServiceCatalogImage = "registry.reg-aws.openshift.com/openshift3/ose-service-catalog:v3.11"
		c.TemplateServiceBrokerImage = "registry.reg-aws.openshift.com/openshift3/ose-template-service-broker:v3.11"
		c.PrometheusNodeExporterImage = "registry.reg-aws.openshift.com/openshift3/prometheus-node-exporter:v0.15.2"
		c.RegistryImage = "registry.reg-aws.openshift.com/openshift3/ose-docker-registry:v3.11"
		c.RouterImage = "registry.reg-aws.openshift.com/openshift3/ose-haproxy-router:v3.11"
		c.RegistryConsoleImage = "registry.reg-aws.openshift.com/openshift3/registry-console:v3.11"
		c.AnsibleServiceBrokerImage = "registry.reg-aws.openshift.com/openshift3/ose-ansible-service-broker:v3.11"
		c.WebConsoleImage = "registry.reg-aws.openshift.com/openshift3/origin-web-console:v3.11"
		c.OAuthProxyImage = "registry.reg-aws.openshift.com/openshift3/oauth-proxy:v3.11"
		c.PrometheusImage = "registry.reg-aws.openshift.com/openshift3/prometheus:v2.2.1"
		c.PrometheusAlertBufferImage = "registry.reg-aws.openshift.com/openshift3/prometheus-alert-buffer:v0.0.2"
		c.PrometheusAlertManagerImage = "registry.reg-aws.openshift.com/openshift3/prometheus-alertmanager:v0.14.0"

		c.TunnelImage = "docker.io/jimminter/tunnel:latest"
		c.SyncImage = "docker.io/jimminter/sync:latest"
		c.AzureCLIImage = "docker.io/microsoft/azure-cli:latest"
	}

}

func selectContainerImages(cs *acsapi.ContainerService, c *Config) {
	switch os.Getenv("DEPLOY_OS") {
	case "":
		selectContainerImagesOSA(cs, c)
	case "centos7":
		selectContainerImagesOrigin(cs, c)
	}
}

func configureDevRegPullSecret(registry string) []byte {
	return []byte(fmt.Sprintf("{\"auths\":{\"%s\":{\"auth\":\"%s\"}}}", registry, os.Getenv("DEV_PULL_SECRET")))
}

func Generate(cs *acsapi.ContainerService, c *Config) (err error) {
	c.Version = versionLatest
	c.TunnelHostname = strings.Replace(cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname, "openshift", "openshift-tunnel", 1)

	selectNodeImage(cs, c)

	selectContainerImages(cs, c)

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
			cn: cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname,
			dnsNames: []string{
				cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname,
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
			cn: cs.Properties.OrchestratorProfile.OpenShiftConfig.RoutingConfigSubdomain,
			dnsNames: []string{
				cs.Properties.OrchestratorProfile.OpenShiftConfig.RoutingConfigSubdomain,
				"*." + cs.Properties.OrchestratorProfile.OpenShiftConfig.RoutingConfigSubdomain,
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.RouterKey,
			cert:        &c.RouterCert,
		},
		{
			cn: "docker-registry-default." + cs.Properties.OrchestratorProfile.OpenShiftConfig.RoutingConfigSubdomain,
			dnsNames: []string{
				"docker-registry-default." + cs.Properties.OrchestratorProfile.OpenShiftConfig.RoutingConfigSubdomain,
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
			endpoint:   cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname,
			username:   "system:admin",
			kubeconfig: &c.AdminKubeconfig,
		},
		{
			clientKey:  c.NodeBootstrapKey,
			clientCert: c.NodeBootstrapCert,
			endpoint:   cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname,
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
