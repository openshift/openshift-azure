package plugin

import (
	"crypto/rsa"
	"crypto/x509"
)

// Config holds the api plugin config structure
type Config struct {
	// SecurityPatchPackages defines a list of rpm packages that fix security issues
	SecurityPatchPackages []string `json:"securityPatchPackages,omitempty"`

	// PluginVersion defines release version of the plugin used to build the cluster
	PluginVersion string `json:"pluginVersion,omitempty"`

	// ComponentLogLevel specifies the log levels for the various openshift components
	ComponentLogLevel ComponentLogLevel `json:"componentLogLevel,omitempty"`

	// SSH to system nodes allowed IP ranges
	SSHSourceAddressPrefixes []string `json:"sshSourceAddressPrefixes,omitempty"`

	Versions map[string]VersionConfig `json:"versions,omitempty"`

	Certificates CertificateConfig `json:"certificates,omitempty"`

	// Geneva Metrics System (MDM) sector used for logging
	GenevaLoggingSector string `json:"genevaLoggingSector,omitempty"`
	// Geneva Metrics System (MDM) logging account
	GenevaLoggingAccount string `json:"genevaLoggingAccount,omitempty"`
	// Geneva Metrics System (MDM) logging namespace
	GenevaLoggingNamespace string `json:"genevaLoggingNamespace,omitempty"`
	// Geneva Metrics System (MDM) logging control plane parameters
	GenevaLoggingControlPlaneAccount     string `json:"genevaLoggingControlPlaneAccount,omitempty"`
	GenevaLoggingControlPlaneEnvironment string `json:"genevaLoggingControlPlaneEnvironment,omitempty"`
	GenevaLoggingControlPlaneRegion      string `json:"genevaLoggingControlPlaneRegion,omitempty"`
	// Geneva Metrics System (MDM) account name for metrics
	GenevaMetricsAccount string `json:"genevaMetricsAccount,omitempty"`
	// Geneva Metrics System (MDM) endpoint for metrics
	GenevaMetricsEndpoint string `json:"genevaMetricsEndpoint,omitempty"`

	// GenevaImagePullSecret defines secret used to pull private Azure images
	GenevaImagePullSecret []byte `json:"genevaImagePullSecret,omitempty"`
	// ImagePullSecret defines the secret used to pull from the private registries, used system-wide
	ImagePullSecret []byte `json:"imagePullSecret,omitempty"`
}

type VersionConfig struct {
	// configuration of VMs in ARM template
	ImageOffer     string `json:"imageOffer,omitempty"`
	ImagePublisher string `json:"imagePublisher,omitempty"`
	ImageSKU       string `json:"imageSku,omitempty"`
	ImageVersion   string `json:"imageVersion,omitempty"`

	Images ImageConfig `json:"images,omitempty"`
}

// ComponentLogLevel represents the log levels for the various components of a
// cluster
type ComponentLogLevel struct {
	APIServer         int `json:"apiServer,omitempty"`
	ControllerManager int `json:"controllerManager,omitempty"`
	Node              int `json:"node,omitempty"`
}

// ImageConfig contains all images for the pods
type ImageConfig struct {
	AlertManager              string `json:"alertManager,omitempty"`
	AnsibleServiceBroker      string `json:"ansibleServiceBroker,omitempty"`
	ClusterMonitoringOperator string `json:"clusterMonitoringOperator,omitempty"`
	ConfigReloader            string `json:"configReloader,omitempty"`
	Console                   string `json:"console,omitempty"`
	ControlPlane              string `json:"controlPlane,omitempty"`
	Grafana                   string `json:"grafana,omitempty"`
	KubeRbacProxy             string `json:"kubeRbacProxy,omitempty"`
	KubeStateMetrics          string `json:"kubeStateMetrics,omitempty"`
	Node                      string `json:"node,omitempty"`
	NodeExporter              string `json:"nodeExporter,omitempty"`
	OAuthProxy                string `json:"oAuthProxy,omitempty"`
	Prometheus                string `json:"prometheus,omitempty"`
	PrometheusConfigReloader  string `json:"prometheusConfigReloader,omitempty"`
	PrometheusOperator        string `json:"prometheusOperator,omitempty"`
	Registry                  string `json:"registry,omitempty"`
	RegistryConsole           string `json:"registryConsole,omitempty"`
	Router                    string `json:"router,omitempty"`
	ServiceCatalog            string `json:"serviceCatalog,omitempty"`
	TemplateServiceBroker     string `json:"templateServiceBroker,omitempty"`
	WebConsole                string `json:"webConsole,omitempty"`

	Format string `json:"format,omitempty"`

	Httpd      string `json:"httpd,omitempty"`
	MasterEtcd string `json:"masterEtcd,omitempty"`

	GenevaLogging string `json:"genevaLogging,omitempty"`
	GenevaStatsd  string `json:"genevaStatsd,omitempty"`
	GenevaTDAgent string `json:"genevaTDAgent,omitempty"`

	AzureControllers string `json:"azureControllers,omitempty"`
	Canary           string `json:"canary,omitempty"`
	EtcdBackup       string `json:"etcdBackup,omitempty"`
	MetricsBridge    string `json:"metricsBridge,omitempty"`
	Startup          string `json:"startup,omitempty"`
	Sync             string `json:"sync,omitempty"`
	TLSProxy         string `json:"tlsProxy,omitempty"`

	MonitorAgent string `json:"monitorAgent,omitempty"`
}

// CertificateConfig contains all certificate configuration for the cluster.
type CertificateConfig struct {
	// geneva integration certificates
	GenevaLogging CertKeyPair `json:"genevaLogging,omitempty"`
	GenevaMetrics CertKeyPair `json:"genevaMetrics,omitempty"`

	// red hat cdn client certificates
	PackageRepository CertKeyPair `json:"packageRepository,omitempty"`
}

// CertKeyPair is an rsa private key and x509 certificate pair.
type CertKeyPair struct {
	Key  *rsa.PrivateKey   `json:"key,omitempty"`
	Cert *x509.Certificate `json:"cert,omitempty"`
}
