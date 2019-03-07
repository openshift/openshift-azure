package validate

import (
	"fmt"
	"net"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

// PluginAPIValidator validator for external Plugin API
type PluginAPIValidator struct{}

// NewPluginAPIValidator return instance of external Plugin API validator
func NewPluginAPIValidator() *PluginAPIValidator {
	return &PluginAPIValidator{}
}

// Validate validates an Plugin API external template/config struct
func (v *PluginAPIValidator) Validate(c *pluginapi.Config) (errs []error) {
	if c == nil {
		errs = append(errs, fmt.Errorf("config cannot be nil"))
		return
	}

	if !rxClusterVersion.MatchString(c.ClusterVersion) {
		errs = append(errs, fmt.Errorf("invalid clusterVersion %q", c.ClusterVersion))
	}

	if c.ImageOffer != "osa" {
		errs = append(errs, fmt.Errorf("invalid imageOffer %q", c.ImageOffer))
	}

	if c.ImagePublisher != "redhat" {
		errs = append(errs, fmt.Errorf("invalid imagePublisher %q", c.ImagePublisher))
	}

	switch c.ImageSKU {
	case "osa_311":
	default:
		errs = append(errs, fmt.Errorf("invalid imageSKU %q", c.ImageSKU))
	}

	if !rxImageVersion.MatchString(c.ImageVersion) {
		errs = append(errs, fmt.Errorf("invalid imageVersion %q", c.ImageVersion))
	}

	for _, prefix := range c.SSHSourceAddressPrefixes {
		if _, _, err := net.ParseCIDR(prefix); err != nil {
			errs = append(errs, fmt.Errorf("invalid sshSourceAddressPrefix %q", prefix))
		}
	}

	errs = append(errs, validateCertificateConfig(&c.Certificates)...)

	errs = append(errs, validateImageConfig(&c.Images)...)

	if c.GenevaLoggingSector == "" {
		errs = append(errs, fmt.Errorf("invalid genevaLoggingSector %q", c.GenevaLoggingSector))
	}

	if c.GenevaLoggingAccount == "" {
		errs = append(errs, fmt.Errorf("invalid genevaLoggingAccount %q", c.GenevaLoggingAccount))
	}

	if c.GenevaLoggingNamespace == "" {
		errs = append(errs, fmt.Errorf("invalid genevaLoggingNamespace %q", c.GenevaLoggingNamespace))
	}

	if c.GenevaLoggingControlPlaneAccount == "" {
		errs = append(errs, fmt.Errorf("invalid genevaLoggingControlPlaneAccount %q", c.GenevaLoggingControlPlaneAccount))
	}

	if c.GenevaLoggingControlPlaneEnvironment == "" {
		errs = append(errs, fmt.Errorf("invalid genevaLoggingControlPlaneEnvironment %q", c.GenevaLoggingControlPlaneEnvironment))
	}

	if c.GenevaLoggingControlPlaneRegion == "" {
		errs = append(errs, fmt.Errorf("invalid genevaLoggingControlPlaneRegion %q", c.GenevaLoggingControlPlaneRegion))
	}

	if c.GenevaMetricsAccount == "" {
		errs = append(errs, fmt.Errorf("invalid genevaMetricsAccount %q", c.GenevaMetricsAccount))
	}

	if c.GenevaMetricsEndpoint == "" {
		errs = append(errs, fmt.Errorf("invalid genevaMetricsEndpoint %q", c.GenevaMetricsEndpoint))
	}

	return
}

func validateImageConfig(i *pluginapi.ImageConfig) (errs []error) {
	if i == nil {
		errs = append(errs, fmt.Errorf("imageConfig cannot be nil"))
		return
	}

	if i.Format == "" {
		errs = append(errs, fmt.Errorf("invalid images.format %q", i.Format))
	}

	if i.ClusterMonitoringOperator == "" {
		errs = append(errs, fmt.Errorf("invalid images.clusterMonitoringOperator %q", i.ClusterMonitoringOperator))
	}

	if i.AzureControllers == "" {
		errs = append(errs, fmt.Errorf("invalid images.azureControllers %q", i.AzureControllers))
	}

	if i.PrometheusOperatorBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.prometheusOperatorBase %q", i.PrometheusOperatorBase))
	}

	if i.PrometheusBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.prometheusBase %q", i.PrometheusBase))
	}

	if i.PrometheusConfigReloaderBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.prometheusConfigReloaderBase %q", i.PrometheusConfigReloaderBase))
	}

	if i.ConfigReloaderBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.configReloaderBase %q", i.ConfigReloaderBase))
	}

	if i.AlertManagerBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.alertManagerBase %q", i.AlertManagerBase))
	}

	if i.NodeExporterBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.nodeExporterBase %q", i.NodeExporterBase))
	}

	if i.GrafanaBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.grafanaBase %q", i.GrafanaBase))
	}

	if i.KubeStateMetricsBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.kubeStateMetricsBase %q", i.KubeStateMetricsBase))
	}

	if i.KubeRbacProxyBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.kubeRbacProxyBase %q", i.KubeRbacProxyBase))
	}

	if i.OAuthProxyBase == "" {
		errs = append(errs, fmt.Errorf("invalid images.oAuthProxyBase %q", i.OAuthProxyBase))
	}

	if i.MasterEtcd == "" {
		errs = append(errs, fmt.Errorf("invalid images.masterEtcd %q", i.MasterEtcd))
	}

	if i.ControlPlane == "" {
		errs = append(errs, fmt.Errorf("invalid images.controlPlane %q", i.ControlPlane))
	}

	if i.Node == "" {
		errs = append(errs, fmt.Errorf("invalid images.node %q", i.Node))
	}

	if i.ServiceCatalog == "" {
		errs = append(errs, fmt.Errorf("invalid images.serviceCatalog %q", i.ServiceCatalog))
	}

	if i.Sync == "" {
		errs = append(errs, fmt.Errorf("invalid images.sync %q", i.Sync))
	}

	if i.Startup == "" {
		errs = append(errs, fmt.Errorf("invalid images.startup %q", i.Startup))
	}

	if i.TemplateServiceBroker == "" {
		errs = append(errs, fmt.Errorf("invalid images.templateServiceBroker %q", i.TemplateServiceBroker))
	}

	if i.Registry == "" {
		errs = append(errs, fmt.Errorf("invalid images.registry %q", i.Registry))
	}

	if i.Router == "" {
		errs = append(errs, fmt.Errorf("invalid images.router %q", i.Router))
	}

	if i.RegistryConsole == "" {
		errs = append(errs, fmt.Errorf("invalid images.registryConsole %q", i.RegistryConsole))
	}

	if i.AnsibleServiceBroker == "" {
		errs = append(errs, fmt.Errorf("invalid images.ansibleServiceBroker %q", i.AnsibleServiceBroker))
	}

	if i.WebConsole == "" {
		errs = append(errs, fmt.Errorf("invalid images.webConsole %q", i.WebConsole))
	}

	if i.Console == "" {
		errs = append(errs, fmt.Errorf("invalid images.console %q", i.Console))
	}

	if i.EtcdBackup == "" {
		errs = append(errs, fmt.Errorf("invalid images.etcdBackup %q", i.EtcdBackup))
	}

	if len(i.GenevaImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("invalid images.genevaImagePullSecret %q", i.GenevaImagePullSecret))
	}

	if i.GenevaLogging == "" {
		errs = append(errs, fmt.Errorf("invalid images.genevaLogging %q", i.GenevaLogging))
	}

	if i.GenevaTDAgent == "" {
		errs = append(errs, fmt.Errorf("invalid images.genevaTDAgent %q", i.GenevaTDAgent))
	}

	if i.GenevaStatsd == "" {
		errs = append(errs, fmt.Errorf("invalid images.genevaStatsd %q", i.GenevaStatsd))
	}

	if i.MetricsBridge == "" {
		errs = append(errs, fmt.Errorf("invalid images.metricsBridge %q", i.MetricsBridge))
	}

	if len(i.ImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("invalid images.imagePullSecret %q", i.ImagePullSecret))
	}

	return
}

func validateCertificateConfig(c *pluginapi.CertificateConfig) (errs []error) {
	if c == nil {
		errs = append(errs, fmt.Errorf("certificateConfig cannot be nil"))
		return
	}

	errs = append(errs, validateCertKeyPair("certificates.genevaLogging", &c.GenevaLogging)...)

	errs = append(errs, validateCertKeyPair("certificates.genevaMetrics", &c.GenevaMetrics)...)

	return
}

func validateCertKeyPair(path string, c *pluginapi.CertKeyPair) (errs []error) {
	if c == nil {
		errs = append(errs, fmt.Errorf("%s cannot be nil", path))
		return
	}

	if c.Key == nil {
		errs = append(errs, fmt.Errorf("invalid %s.key", path))
	} else if err := c.Key.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid %s.key: %v", path, err))
	}

	if c.Cert == nil {
		errs = append(errs, fmt.Errorf("invalid %s.cert", path))
	}

	return errs
}
