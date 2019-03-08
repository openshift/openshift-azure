package api

import (
	plugin "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

// ConvertFromPlugin converts external plugin API config type into
// internal API Config type
func ConvertFromPlugin(in *plugin.Config, old *Config) *Config {
	cs := &Config{}
	if old != nil {
		cs = old.DeepCopy()
	}
	cs.PluginVersion = in.PluginVersion
	// Generic offering configurables
	cs.ImageOffer = in.ImageOffer
	cs.ImagePublisher = in.ImagePublisher
	cs.ImageSKU = in.ImageSKU
	cs.ImageVersion = in.ImageVersion
	cs.SSHSourceAddressPrefixes = in.SSHSourceAddressPrefixes
	// Geneva integration configurables
	cs.GenevaLoggingSector = in.GenevaLoggingSector
	cs.GenevaLoggingNamespace = in.GenevaLoggingNamespace
	cs.GenevaLoggingAccount = in.GenevaLoggingAccount
	cs.GenevaMetricsAccount = in.GenevaMetricsAccount
	cs.GenevaMetricsEndpoint = in.GenevaMetricsEndpoint
	cs.GenevaLoggingControlPlaneAccount = in.GenevaLoggingControlPlaneAccount
	cs.GenevaLoggingControlPlaneEnvironment = in.GenevaLoggingControlPlaneEnvironment
	cs.GenevaLoggingControlPlaneRegion = in.GenevaLoggingControlPlaneRegion
	cs.Certificates.GenevaLogging.Cert = in.Certificates.GenevaLogging.Cert
	cs.Certificates.GenevaLogging.Key = in.Certificates.GenevaLogging.Key
	cs.Certificates.GenevaMetrics.Cert = in.Certificates.GenevaMetrics.Cert
	cs.Certificates.GenevaMetrics.Key = in.Certificates.GenevaMetrics.Key
	// Container images configuration
	cs.Images.ImagePullSecret = in.Images.ImagePullSecret
	cs.Images.GenevaImagePullSecret = in.Images.GenevaImagePullSecret
	cs.Images.Format = in.Images.Format
	cs.Images.ClusterMonitoringOperator = in.Images.ClusterMonitoringOperator
	cs.Images.AzureControllers = in.Images.AzureControllers
	cs.Images.PrometheusOperator = in.Images.PrometheusOperator
	cs.Images.Prometheus = in.Images.Prometheus
	cs.Images.PrometheusConfigReloader = in.Images.PrometheusConfigReloader
	cs.Images.ConfigReloader = in.Images.ConfigReloader
	cs.Images.AlertManager = in.Images.AlertManager
	cs.Images.NodeExporter = in.Images.NodeExporter
	cs.Images.Grafana = in.Images.Grafana
	cs.Images.KubeStateMetrics = in.Images.KubeStateMetrics
	cs.Images.KubeRbacProxy = in.Images.KubeRbacProxy
	cs.Images.OAuthProxy = in.Images.OAuthProxy
	cs.Images.MasterEtcd = in.Images.MasterEtcd
	cs.Images.ControlPlane = in.Images.ControlPlane
	cs.Images.Node = in.Images.Node
	cs.Images.ServiceCatalog = in.Images.ServiceCatalog
	cs.Images.Sync = in.Images.Sync
	cs.Images.TemplateServiceBroker = in.Images.TemplateServiceBroker
	cs.Images.Registry = in.Images.Registry
	cs.Images.Router = in.Images.Router
	cs.Images.RegistryConsole = in.Images.RegistryConsole
	cs.Images.AnsibleServiceBroker = in.Images.AnsibleServiceBroker
	cs.Images.WebConsole = in.Images.WebConsole
	cs.Images.Console = in.Images.Console
	cs.Images.EtcdBackup = in.Images.EtcdBackup
	cs.Images.Httpd = in.Images.Httpd
	cs.Images.Startup = in.Images.Startup
	cs.Images.GenevaLogging = in.Images.GenevaLogging
	cs.Images.GenevaTDAgent = in.Images.GenevaTDAgent
	cs.Images.GenevaStatsd = in.Images.GenevaStatsd
	cs.Images.MetricsBridge = in.Images.MetricsBridge

	return cs
}
