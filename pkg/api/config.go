package api

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/satori/go.uuid"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

type Config struct {
	Version int

	ImageOffer     string
	ImagePublisher string
	ImageSKU       string
	ImageVersion   string

	// for development
	ImageResourceGroup string
	ImageResourceName  string

	Certificates CertificateConfig

	// container images for pods
	MasterEtcdImage             string
	ControlPlaneImage           string
	NodeImage                   string
	ServiceCatalogImage         string
	SyncImage                   string
	TemplateServiceBrokerImage  string
	PrometheusNodeExporterImage string
	RegistryImage               string
	RouterImage                 string
	AzureCLIImage               string
	RegistryConsoleImage        string
	AnsibleServiceBrokerImage   string
	WebConsoleImage             string
	OAuthProxyImage             string
	PrometheusImage             string
	PrometheusAlertBufferImage  string
	PrometheusAlertManagerImage string

	// kubeconfigs
	AdminKubeconfig              *v1.Config
	MasterKubeconfig             *v1.Config
	NodeBootstrapKubeconfig      *v1.Config
	AzureClusterReaderKubeconfig *v1.Config

	// misc control plane configurables
	ServiceAccountKey *rsa.PrivateKey
	SessionSecretAuth []byte
	SessionSecretEnc  []byte
	HtPasswd          []byte
	ImageConfigFormat string

	// misc node configurables
	SSHKey *rsa.PrivateKey

	// misc infra configurables
	RegistryHTTPSecret             []byte
	AlertManagerProxySessionSecret []byte
	AlertsProxySessionSecret       []byte
	PrometheusProxySessionSecret   []byte
	ServiceCatalogClusterID        uuid.UUID
	// random string based configurables
	RegistryStorageAccount     string
	RegistryConsoleOAuthSecret string
	RouterStatsPassword        string

	// DNS configurables
	RouterLBCNamePrefix string
	MasterLBCNamePrefix string

	// enriched values which are not present in the external API representation
	TenantID       string
	SubscriptionID string
	ResourceGroup  string

	CloudProviderConf []byte

	ConfigStorageAccount string
}

// CertificateConfig contains all certificate configuration for the cluster.
type CertificateConfig struct {
	// CAs
	EtcdCa           CertKeyPair
	Ca               CertKeyPair
	FrontProxyCa     CertKeyPair
	ServiceSigningCa CertKeyPair
	ServiceCatalogCa CertKeyPair

	// etcd certificates
	EtcdServer CertKeyPair
	EtcdPeer   CertKeyPair
	EtcdClient CertKeyPair

	// control plane certificates
	MasterServer         CertKeyPair
	OpenshiftConsole     CertKeyPair
	Admin                CertKeyPair
	AggregatorFrontProxy CertKeyPair
	MasterKubeletClient  CertKeyPair
	MasterProxyClient    CertKeyPair
	OpenShiftMaster      CertKeyPair
	NodeBootstrap        CertKeyPair

	// infra certificates
	Registry                CertKeyPair
	Router                  CertKeyPair
	ServiceCatalogServer    CertKeyPair
	ServiceCatalogAPIClient CertKeyPair

	// misc certificates
	AzureClusterReader CertKeyPair
}

// CertKeyPair is an rsa private key and x509 certificate pair.
type CertKeyPair struct {
	Key  *rsa.PrivateKey
	Cert *x509.Certificate
}
