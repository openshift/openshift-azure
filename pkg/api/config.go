package api

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/satori/go.uuid"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

type Config struct {
	ImageOffer     string `json:"imageOffer,omitempty"`
	ImagePublisher string `json:"imagePublisher,omitempty"`
	ImageSKU       string `json:"imageSku,omitempty"`
	ImageVersion   string `json:"imageVersion,omitempty"`

	// for development
	ImageResourceGroup string `json:"imageResourceGroup,omitempty"`
	ImageResourceName  string `json:"imageResourceName,omitempty"`

	Certificates CertificateConfig `json:"certificates,omitempty"`

	// container images for pods
	MasterEtcdImage             string `json:"masterEtcdImage,omitempty"`
	ControlPlaneImage           string `json:"controlPlaneImage,omitempty"`
	NodeImage                   string `json:"nodeImage,omitempty"`
	ServiceCatalogImage         string `json:"serviceCatalogImage,omitempty"`
	SyncImage                   string `json:"syncImage,omitempty"`
	TemplateServiceBrokerImage  string `json:"templateServiceBrokerImage,omitempty"`
	PrometheusNodeExporterImage string `json:"prometheusNodeExporterImage,omitempty"`
	RegistryImage               string `json:"registryImage,omitempty"`
	RouterImage                 string `json:"routerImage,omitempty"`
	AzureCLIImage               string `json:"azureCliImage,omitempty"`
	RegistryConsoleImage        string `json:"registryConsoleImage,omitempty"`
	AnsibleServiceBrokerImage   string `json:"ansibleServiceBrokerImage,omitempty"`
	WebConsoleImage             string `json:"webConsoleImage,omitempty"`
	OAuthProxyImage             string `json:"oAuthProxyImage,omitempty"`
	PrometheusImage             string `json:"prometheusImage,omitempty"`
	PrometheusAlertBufferImage  string `json:"prometheusAlertBufferImage,omitempty"`
	PrometheusAlertManagerImage string `json:"prometheusAlertManagerImage,omitempty"`
	LogBridgeImage              string `json:"logBridgeImage,omitempty"`
	EtcdOperatorImage           string `json:"etcdOperatorImage,omitempty"`

	// kubeconfigs
	AdminKubeconfig              *v1.Config `json:"adminKubeconfig,omitempty"`
	MasterKubeconfig             *v1.Config `json:"masterKubeconfig,omitempty"`
	NodeBootstrapKubeconfig      *v1.Config `json:"nodeBootstrapKubeconfig,omitempty"`
	AzureClusterReaderKubeconfig *v1.Config `json:"azureClusterReaderKubeconfig,omitempty"`

	// misc control plane configurables
	ServiceAccountKey *rsa.PrivateKey `json:"serviceAccountKey,omitempty"`
	SessionSecretAuth []byte          `json:"sessionSecretAuth,omitempty"`
	SessionSecretEnc  []byte          `json:"sessionSecretEnc,omitempty"`
	HtPasswd          []byte          `json:"htPasswd,omitempty"`
	//TODO: Remove me before GA!
	AdminPasswd       string `json:"adminPasswd,omitempty"`
	ImageConfigFormat string `json:"imageConfigFormat,omitempty"`

	// misc node configurables
	SSHKey *rsa.PrivateKey `json:"sshKey,omitempty"`

	// misc infra configurables
	RegistryHTTPSecret             []byte    `json:"registryHttpSecret,omitempty"`
	AlertManagerProxySessionSecret []byte    `json:"alertManagerProxySessionSecret,omitempty"`
	AlertsProxySessionSecret       []byte    `json:"alertsProxySessionSecret,omitempty"`
	PrometheusProxySessionSecret   []byte    `json:"prometheusProxySessionSecret,omitempty"`
	ServiceCatalogClusterID        uuid.UUID `json:"serviceCatalogClusterId,omitempty"`
	// random string based configurables
	RegistryStorageAccount     string `json:"registryStorageAccount,omitempty"`
	RegistryConsoleOAuthSecret string `json:"registryConsoleOAuthSecret,omitempty"`
	RouterStatsPassword        string `json:"routerStatsPassword,omitempty"`
	LoggingWorkspace           string `json:"loggingWorkspace,omitempty"` // workspace for Azure Log Analytics resource

	// DNS configurables
	MasterLBCNamePrefix string `json:"masterLbCNamePrefix,omitempty"`

	// enriched values which are not present in the external API representation
	TenantID       string `json:"tenantId,omitempty"`
	SubscriptionID string `json:"subscriptionId,omitempty"`
	ResourceGroup  string `json:"resourceGroup,omitempty"`

	CloudProviderConf []byte `json:"cloudProviderConf,omitempty"`

	ConfigStorageAccount string `json:"configStorageAccount,omitempty"`
}

// CertificateConfig contains all certificate configuration for the cluster.
type CertificateConfig struct {
	// CAs
	EtcdCa           CertKeyPair `json:"etcdCa,omitempty"`
	Ca               CertKeyPair `json:"ca,omitempty"`
	FrontProxyCa     CertKeyPair `json:"frontProxyCa,omitempty"`
	ServiceSigningCa CertKeyPair `json:"serviceSigningCa,omitempty"`
	ServiceCatalogCa CertKeyPair `json:"serviceCatalogCa,omitempty"`

	// etcd certificates
	EtcdServer CertKeyPair `json:"etcdServer,omitempty"`
	EtcdPeer   CertKeyPair `json:"etcdPeer,omitempty"`
	EtcdClient CertKeyPair `json:"etcdClient,omitempty"`

	// control plane certificates
	MasterServer         CertKeyPair `json:"masterServer,omitempty"`
	OpenshiftConsole     CertKeyPair `json:"openshiftConsole,omitempty"`
	Admin                CertKeyPair `json:"admin,omitempty"`
	AggregatorFrontProxy CertKeyPair `json:"aggregatorFrontProxy,omitempty"`
	MasterKubeletClient  CertKeyPair `json:"masterKubeletClient,omitempty"`
	MasterProxyClient    CertKeyPair `json:"masterProxyClient,omitempty"`
	OpenShiftMaster      CertKeyPair `json:"openShiftMaster,omitempty"`
	NodeBootstrap        CertKeyPair `json:"nodeBootstrap,omitempty"`

	// infra certificates
	Registry                CertKeyPair `json:"registry,omitempty"`
	Router                  CertKeyPair `json:"router,omitempty"`
	ServiceCatalogServer    CertKeyPair `json:"serviceCatalogServer,omitempty"`
	ServiceCatalogAPIClient CertKeyPair `json:"serviceCatalogAPIClient,omitempty"`

	// misc certificates
	AzureClusterReader CertKeyPair `json:"azureClusterReader,omitempty"`
}

// CertKeyPair is an rsa private key and x509 certificate pair.
type CertKeyPair struct {
	Key  *rsa.PrivateKey   `json:"key,omitempty"`
	Cert *x509.Certificate `json:"cert,omitempty"`
}
