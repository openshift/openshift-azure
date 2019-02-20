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
	cs.ClusterVersion = in.ClusterVersion
	cs.ClusterLogLevel.ApiServer = in.ClusterLogLevel.ApiServer
	cs.ClusterLogLevel.ControllerManager = in.ClusterLogLevel.ControllerManager
	cs.ClusterLogLevel.Node = in.ClusterLogLevel.Node
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
	cs.Images.PrometheusOperatorBase = in.Images.PrometheusOperatorBase
	cs.Images.PrometheusBase = in.Images.PrometheusBase
	cs.Images.PrometheusConfigReloaderBase = in.Images.PrometheusConfigReloaderBase
	cs.Images.ConfigReloaderBase = in.Images.ConfigReloaderBase
	cs.Images.AlertManagerBase = in.Images.AlertManagerBase
	cs.Images.NodeExporterBase = in.Images.NodeExporterBase
	cs.Images.GrafanaBase = in.Images.GrafanaBase
	cs.Images.KubeStateMetricsBase = in.Images.KubeStateMetricsBase
	cs.Images.KubeRbacProxyBase = in.Images.KubeRbacProxyBase
	cs.Images.OAuthProxyBase = in.Images.OAuthProxyBase
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
	cs.Images.Startup = in.Images.Startup
	cs.Images.GenevaLogging = in.Images.GenevaLogging
	cs.Images.GenevaTDAgent = in.Images.GenevaTDAgent
	cs.Images.GenevaStatsd = in.Images.GenevaStatsd
	cs.Images.MetricsBridge = in.Images.MetricsBridge

	return cs
}
