package api

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/satori/go.uuid"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

type Config struct {
	// configuration of VMs in ARM template
	ImageOffer     string `json:"imageOffer,omitempty"`
	ImagePublisher string `json:"imagePublisher,omitempty"`
	ImageSKU       string `json:"imageSku,omitempty"`
	ImageVersion   string `json:"imageVersion,omitempty"`

	SSHKey *rsa.PrivateKey `json:"sshKey,omitempty"`

	// configuration of other ARM resources
	ConfigStorageAccount   string `json:"configStorageAccount,omitempty"`
	RegistryStorageAccount string `json:"registryStorageAccount,omitempty"`
	LoggingWorkspace       string `json:"loggingWorkspace,omitempty"` // workspace for Azure Log Analytics resource
	LoggingLocation        string `json:"loggingLocation,omitempty"`  // location for Azure Log Analytics resource

	Certificates CertificateConfig `json:"certificates,omitempty"`
	Images       ImageConfig       `json:"images,omitempty"`

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
	AdminPasswd       string          `json:"adminPasswd,omitempty"` //TODO: Remove me before GA!

	// misc infra configurables
	RegistryHTTPSecret             []byte    `json:"registryHttpSecret,omitempty"`
	PrometheusProxySessionSecret   []byte    `json:"prometheusProxySessionSecret,omitempty"`
	AlertManagerProxySessionSecret []byte    `json:"alertManagerProxySessionSecret,omitempty"`
	AlertsProxySessionSecret       []byte    `json:"alertsProxySessionSecret,omitempty"`
	RegistryConsoleOAuthSecret     string    `json:"registryConsoleOAuthSecret,omitempty"`
	RouterStatsPassword            string    `json:"routerStatsPassword,omitempty"`
	ServiceCatalogClusterID        uuid.UUID `json:"serviceCatalogClusterId,omitempty"`
}

// ImageConfig contains all images for the pods
type ImageConfig struct {
	MasterEtcd             string `json:"masterEtcd,omitempty"`
	ControlPlane           string `json:"controlPlane,omitempty"`
	Node                   string `json:"node,omitempty"`
	ServiceCatalog         string `json:"serviceCatalog,omitempty"`
	Sync                   string `json:"sync,omitempty"`
	TemplateServiceBroker  string `json:"templateServiceBroker,omitempty"`
	PrometheusNodeExporter string `json:"prometheusNodeExporter,omitempty"`
	Registry               string `json:"registry,omitempty"`
	Router                 string `json:"router,omitempty"`
	RegistryConsole        string `json:"registryConsole,omitempty"`
	AnsibleServiceBroker   string `json:"ansibleServiceBroker,omitempty"`
	WebConsole             string `json:"webConsole,omitempty"`
	OAuthProxy             string `json:"oAuthProxy,omitempty"`
	Prometheus             string `json:"prometheus,omitempty"`
	PrometheusAlertBuffer  string `json:"prometheusAlertBuffer,omitempty"`
	PrometheusAlertManager string `json:"prometheusAlertManager,omitempty"`
	LogBridge              string `json:"logBridge,omitempty"`
	EtcdOperator           string `json:"etcdOperator,omitempty"`
	KubeStateMetrics       string `json:"kubeStateMetrics,omitempty"`
	AddonsResizer          string `json:"addonsResizer,omitempty"`
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
