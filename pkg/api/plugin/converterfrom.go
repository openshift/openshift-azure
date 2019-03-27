package plugin

import (
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
)

// ConvertFrom converts from a
// plugin.OpenShiftManagedCluster.Config to an internal.OpenShiftManagedCluster.Config
// If old is non-nil, it is going to be used as the base for the internal
// output where the external request is merged on top of.
func ConvertFrom(in *Config, old *api.Config, version string) (*api.Config, error) {
	if _, found := in.Versions[version]; !found {
		return nil, fmt.Errorf("version %q not found", version)
	}
	c := &api.Config{}
	if old != nil {
		c = old.DeepCopy()
	}

	// do not set c.PluginVersion = in.PluginVersion: this is decided by the
	// upgrade code!

	if c.ComponentLogLevel.APIServer == nil {
		c.ComponentLogLevel.APIServer = &in.ComponentLogLevel.APIServer
	}
	if c.ComponentLogLevel.ControllerManager == nil {
		c.ComponentLogLevel.ControllerManager = &in.ComponentLogLevel.ControllerManager
	}
	if c.ComponentLogLevel.Node == nil {
		c.ComponentLogLevel.Node = &in.ComponentLogLevel.Node
	}

	c.SSHSourceAddressPrefixes = in.SSHSourceAddressPrefixes

	// Generic offering configurables
	c.ImageOffer = in.Versions[version].ImageOffer
	c.ImagePublisher = in.Versions[version].ImagePublisher
	c.ImageSKU = in.Versions[version].ImageSKU
	c.ImageVersion = in.Versions[version].ImageVersion

	// Container images configuration
	c.Images.AlertManager = in.Versions[version].Images.AlertManager
	c.Images.AnsibleServiceBroker = in.Versions[version].Images.AnsibleServiceBroker
	c.Images.ClusterMonitoringOperator = in.Versions[version].Images.ClusterMonitoringOperator
	c.Images.ConfigReloader = in.Versions[version].Images.ConfigReloader
	c.Images.Console = in.Versions[version].Images.Console
	c.Images.ControlPlane = in.Versions[version].Images.ControlPlane
	c.Images.Grafana = in.Versions[version].Images.Grafana
	c.Images.KubeRbacProxy = in.Versions[version].Images.KubeRbacProxy
	c.Images.KubeStateMetrics = in.Versions[version].Images.KubeStateMetrics
	c.Images.Node = in.Versions[version].Images.Node
	c.Images.NodeExporter = in.Versions[version].Images.NodeExporter
	c.Images.OAuthProxy = in.Versions[version].Images.OAuthProxy
	c.Images.Prometheus = in.Versions[version].Images.Prometheus
	c.Images.PrometheusConfigReloader = in.Versions[version].Images.PrometheusConfigReloader
	c.Images.PrometheusOperator = in.Versions[version].Images.PrometheusOperator
	c.Images.Registry = in.Versions[version].Images.Registry
	c.Images.RegistryConsole = in.Versions[version].Images.RegistryConsole
	c.Images.Router = in.Versions[version].Images.Router
	c.Images.ServiceCatalog = in.Versions[version].Images.ServiceCatalog
	c.Images.TemplateServiceBroker = in.Versions[version].Images.TemplateServiceBroker
	c.Images.WebConsole = in.Versions[version].Images.WebConsole

	c.Images.Format = in.Versions[version].Images.Format

	c.Images.Httpd = in.Versions[version].Images.Httpd
	c.Images.MasterEtcd = in.Versions[version].Images.MasterEtcd

	c.Images.GenevaLogging = in.Versions[version].Images.GenevaLogging
	c.Images.GenevaStatsd = in.Versions[version].Images.GenevaStatsd
	c.Images.GenevaTDAgent = in.Versions[version].Images.GenevaTDAgent

	c.Images.AzureControllers = in.Versions[version].Images.AzureControllers
	c.Images.Canary = in.Versions[version].Images.Canary
	c.Images.EtcdBackup = in.Versions[version].Images.EtcdBackup
	c.Images.MetricsBridge = in.Versions[version].Images.MetricsBridge
	c.Images.Startup = in.Versions[version].Images.Startup
	c.Images.Sync = in.Versions[version].Images.Sync
	c.Images.TLSProxy = in.Versions[version].Images.TLSProxy

	c.Certificates.GenevaLogging.Key = in.Certificates.GenevaLogging.Key
	c.Certificates.GenevaLogging.Cert = in.Certificates.GenevaLogging.Cert
	c.Certificates.GenevaMetrics.Key = in.Certificates.GenevaMetrics.Key
	c.Certificates.GenevaMetrics.Cert = in.Certificates.GenevaMetrics.Cert

	// Geneva integration configurables
	c.GenevaLoggingSector = in.GenevaLoggingSector
	c.GenevaLoggingAccount = in.GenevaLoggingAccount
	c.GenevaLoggingNamespace = in.GenevaLoggingNamespace
	c.GenevaLoggingControlPlaneAccount = in.GenevaLoggingControlPlaneAccount
	c.GenevaLoggingControlPlaneEnvironment = in.GenevaLoggingControlPlaneEnvironment
	c.GenevaLoggingControlPlaneRegion = in.GenevaLoggingControlPlaneRegion
	c.GenevaMetricsAccount = in.GenevaMetricsAccount
	c.GenevaMetricsEndpoint = in.GenevaMetricsEndpoint

	c.Images.ImagePullSecret = in.ImagePullSecret
	c.Images.GenevaImagePullSecret = in.GenevaImagePullSecret

	return c, nil
}
