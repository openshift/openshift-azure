package plugin

import (
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
)

// ToInternal converts from a plugin.Config to an internal.Config.  If old is
// non-nil, it is going to be used as the base for the internal output where the
// external request is merged on top of.  The argument setVersionFields is used
// on cluster creation or upgrade (not update) to indicate that we should
// overwrite version-related fields from the plugin config.
func ToInternal(in *Config, old *api.Config, setVersionFields bool) (*api.Config, error) {
	c := &api.Config{}
	if old != nil {
		c = old.DeepCopy()
	}

	// setting c.PluginVersion = in.PluginVersion is done up-front by the plugin
	// code.  It could be done here as well (gated by setVersionFields) but
	// would/should be a no-op.  To simplify the logic, we don't do it.

	c.ComponentLogLevel.APIServer = &in.ComponentLogLevel.APIServer
	c.ComponentLogLevel.ControllerManager = &in.ComponentLogLevel.ControllerManager
	c.ComponentLogLevel.Node = &in.ComponentLogLevel.Node

	c.SecurityPatchPackages = in.SecurityPatchPackages
	c.SSHSourceAddressPrefixes = in.SSHSourceAddressPrefixes

	if setVersionFields {
		inVersion, found := in.Versions[c.PluginVersion]
		if !found {
			return nil, fmt.Errorf("version %q not found", c.PluginVersion)
		}

		// Generic offering configurables
		c.ImageOffer = inVersion.ImageOffer
		c.ImagePublisher = inVersion.ImagePublisher
		c.ImageSKU = inVersion.ImageSKU
		c.ImageVersion = inVersion.ImageVersion

		// Container images configuration
		c.Images.AlertManager = inVersion.Images.AlertManager
		c.Images.AnsibleServiceBroker = inVersion.Images.AnsibleServiceBroker
		c.Images.ClusterMonitoringOperator = inVersion.Images.ClusterMonitoringOperator
		c.Images.ConfigReloader = inVersion.Images.ConfigReloader
		c.Images.Console = inVersion.Images.Console
		c.Images.ControlPlane = inVersion.Images.ControlPlane
		c.Images.Grafana = inVersion.Images.Grafana
		c.Images.KubeRbacProxy = inVersion.Images.KubeRbacProxy
		c.Images.KubeStateMetrics = inVersion.Images.KubeStateMetrics
		c.Images.Node = inVersion.Images.Node
		c.Images.NodeExporter = inVersion.Images.NodeExporter
		c.Images.OAuthProxy = inVersion.Images.OAuthProxy
		c.Images.Prometheus = inVersion.Images.Prometheus
		c.Images.PrometheusConfigReloader = inVersion.Images.PrometheusConfigReloader
		c.Images.PrometheusOperator = inVersion.Images.PrometheusOperator
		c.Images.Registry = inVersion.Images.Registry
		c.Images.RegistryConsole = inVersion.Images.RegistryConsole
		c.Images.Router = inVersion.Images.Router
		c.Images.ServiceCatalog = inVersion.Images.ServiceCatalog
		c.Images.TemplateServiceBroker = inVersion.Images.TemplateServiceBroker
		c.Images.WebConsole = inVersion.Images.WebConsole

		c.Images.Format = inVersion.Images.Format

		c.Images.Httpd = inVersion.Images.Httpd
		c.Images.MasterEtcd = inVersion.Images.MasterEtcd

		c.Images.GenevaLogging = inVersion.Images.GenevaLogging
		c.Images.GenevaStatsd = inVersion.Images.GenevaStatsd
		c.Images.GenevaTDAgent = inVersion.Images.GenevaTDAgent

		c.Images.AzureControllers = inVersion.Images.AzureControllers
		c.Images.Canary = inVersion.Images.Canary
		c.Images.AroAdmissionController = inVersion.Images.AroAdmissionController
		c.Images.EtcdBackup = inVersion.Images.EtcdBackup
		c.Images.MetricsBridge = inVersion.Images.MetricsBridge
		c.Images.Startup = inVersion.Images.Startup
		c.Images.Sync = inVersion.Images.Sync
		c.Images.TLSProxy = inVersion.Images.TLSProxy

		c.Images.LogAnalyticsAgent = inVersion.Images.LogAnalyticsAgent
	}

	// use setVersionFields to override the secrets below otherwise
	// they become un-updatable..
	if c.Certificates.GenevaLogging.Key == nil || setVersionFields {
		c.Certificates.GenevaLogging.Key = in.Certificates.GenevaLogging.Key
	}
	if c.Certificates.GenevaLogging.Cert == nil || setVersionFields {
		c.Certificates.GenevaLogging.Cert = in.Certificates.GenevaLogging.Cert
	}
	if c.Certificates.GenevaMetrics.Key == nil || setVersionFields {
		c.Certificates.GenevaMetrics.Key = in.Certificates.GenevaMetrics.Key
	}
	if c.Certificates.GenevaMetrics.Cert == nil || setVersionFields {
		c.Certificates.GenevaMetrics.Cert = in.Certificates.GenevaMetrics.Cert
	}
	if c.Certificates.PackageRepository.Key == nil || setVersionFields {
		c.Certificates.PackageRepository.Key = in.Certificates.PackageRepository.Key
	}
	if c.Certificates.PackageRepository.Cert == nil || setVersionFields {
		c.Certificates.PackageRepository.Cert = in.Certificates.PackageRepository.Cert
	}

	// Geneva integration configurables
	if c.GenevaLoggingSector == "" {
		c.GenevaLoggingSector = in.GenevaLoggingSector
	}
	if c.GenevaLoggingAccount == "" {
		c.GenevaLoggingAccount = in.GenevaLoggingAccount
	}
	if c.GenevaLoggingNamespace == "" {
		c.GenevaLoggingNamespace = in.GenevaLoggingNamespace
	}
	if c.GenevaLoggingControlPlaneAccount == "" {
		c.GenevaLoggingControlPlaneAccount = in.GenevaLoggingControlPlaneAccount
	}
	if c.GenevaLoggingControlPlaneEnvironment == "" {
		c.GenevaLoggingControlPlaneEnvironment = in.GenevaLoggingControlPlaneEnvironment
	}
	if c.GenevaLoggingControlPlaneRegion == "" {
		c.GenevaLoggingControlPlaneRegion = in.GenevaLoggingControlPlaneRegion
	}
	if c.GenevaMetricsAccount == "" {
		c.GenevaMetricsAccount = in.GenevaMetricsAccount
	}
	if c.GenevaMetricsEndpoint == "" {
		c.GenevaMetricsEndpoint = in.GenevaMetricsEndpoint
	}

	if c.Images.ImagePullSecret == nil || setVersionFields {
		c.Images.ImagePullSecret = in.ImagePullSecret
	}
	if c.Images.GenevaImagePullSecret == nil || setVersionFields {
		c.Images.GenevaImagePullSecret = in.GenevaImagePullSecret
	}

	return c, nil
}
