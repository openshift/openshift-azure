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
		errs = append(errs, fmt.Errorf("Config cannot be nil"))
		return
	}

	if !rxClusterVersion.MatchString(c.ClusterVersion) {
		errs = append(errs, fmt.Errorf("invalid ClusterVersion %q", c.ClusterVersion))
	}

	if c.ImageOffer != "osa" {
		errs = append(errs, fmt.Errorf("imageOffer should be osa"))
	}

	if c.ImagePublisher != "redhat" {
		errs = append(errs, fmt.Errorf("imagePublisher should be redhat"))
	}

	switch c.ImageSKU {
	case "osa_311":
	default:
		errs = append(errs, fmt.Errorf("invalid ImageSKU %q", c.ImageSKU))
	}

	if !rxImageVersion.MatchString(c.ImageVersion) {
		errs = append(errs, fmt.Errorf("invalid ImageVersion %q", c.ImageVersion))
	}

	for _, prefix := range c.SSHSourceAddressPrefixes {
		if _, _, err := net.ParseCIDR(prefix); err != nil {
			errs = append(errs, fmt.Errorf("invalid sshSourceAddressPrefix %q", prefix))
		}
	}

	errs = append(errs, v.validateCertificateConfig(&c.Certificates)...)

	errs = append(errs, v.validateImageConfig(&c.Images)...)

	if c.GenevaLoggingSector == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingSector cannot be empty"))
	}

	if c.GenevaLoggingAccount == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingAccount cannot be empty"))
	}

	if c.GenevaLoggingNamespace == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingNamespace cannot be empty"))
	}

	if c.GenevaLoggingControlPlaneAccount == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingControlPlaneAccount cannot be empty"))
	}

	if c.GenevaLoggingControlPlaneEnvironment == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingControlPlaneEnvironment cannot be empty"))
	}

	if c.GenevaLoggingControlPlaneRegion == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingControlPlaneRegion cannot be empty"))
	}

	if c.GenevaMetricsAccount == "" {
		errs = append(errs, fmt.Errorf("genevaMetricsAccount cannot be empty"))
	}

	if c.GenevaMetricsEndpoint == "" {
		errs = append(errs, fmt.Errorf("genevaMetricsEndpoint cannot be empty"))
	}

	return
}

func (v *PluginAPIValidator) validateImageConfig(i *pluginapi.ImageConfig) (errs []error) {
	if i == nil {
		errs = append(errs, fmt.Errorf("imageConfig cannot be nil"))
		return
	}

	if i.Format == "" {
		errs = append(errs, fmt.Errorf("images.Format cannot be empty"))
	}

	if i.ClusterMonitoringOperator == "" {
		errs = append(errs, fmt.Errorf("images.ClusterMonitoringOperator cannot be empty"))
	}

	if i.AzureControllers == "" {
		errs = append(errs, fmt.Errorf("images.AzureControllers cannot be empty"))
	}

	if i.PrometheusOperatorBase == "" {
		errs = append(errs, fmt.Errorf("images.PrometheusOperatorBase cannot be empty"))
	}

	if i.PrometheusBase == "" {
		errs = append(errs, fmt.Errorf("images.PrometheusBase cannot be empty"))
	}

	if i.PrometheusConfigReloaderBase == "" {
		errs = append(errs, fmt.Errorf("images.PrometheusConfigReloaderBase cannot be empty"))
	}

	if i.ConfigReloaderBase == "" {
		errs = append(errs, fmt.Errorf("images.ConfigReloaderBase cannot be empty"))
	}

	if i.AlertManagerBase == "" {
		errs = append(errs, fmt.Errorf("images.AlertManagerBase cannot be empty"))
	}

	if i.NodeExporterBase == "" {
		errs = append(errs, fmt.Errorf("images.NodeExporterBase cannot be empty"))
	}

	if i.GrafanaBase == "" {
		errs = append(errs, fmt.Errorf("images.GrafanaBase cannot be empty"))
	}

	if i.KubeStateMetricsBase == "" {
		errs = append(errs, fmt.Errorf("images.KubeStateMetricsBase cannot be empty"))
	}

	if i.KubeRbacProxyBase == "" {
		errs = append(errs, fmt.Errorf("images.KubeRbacProxyBase cannot be empty"))
	}

	if i.OAuthProxyBase == "" {
		errs = append(errs, fmt.Errorf("images.OAuthProxyBase cannot be empty"))
	}

	if i.MasterEtcd == "" {
		errs = append(errs, fmt.Errorf("images.MasterEtcd cannot be empty"))
	}

	if i.ControlPlane == "" {
		errs = append(errs, fmt.Errorf("images.ControlPlane cannot be empty"))
	}

	if i.Node == "" {
		errs = append(errs, fmt.Errorf("images.Node cannot be empty"))
	}

	if i.ServiceCatalog == "" {
		errs = append(errs, fmt.Errorf("images.ServiceCatalog cannot be empty"))
	}

	if i.Sync == "" {
		errs = append(errs, fmt.Errorf("images.Sync cannot be empty"))
	}

	if i.Startup == "" {
		errs = append(errs, fmt.Errorf("images.Startup cannot be empty"))
	}

	if i.TemplateServiceBroker == "" {
		errs = append(errs, fmt.Errorf("images.TemplateServiceBroker cannot be empty"))
	}

	if i.Registry == "" {
		errs = append(errs, fmt.Errorf("images.Registry cannot be empty"))
	}

	if i.Router == "" {
		errs = append(errs, fmt.Errorf("images.Router cannot be empty"))
	}

	if i.RegistryConsole == "" {
		errs = append(errs, fmt.Errorf("images.RegistryConsole cannot be empty"))
	}

	if i.AnsibleServiceBroker == "" {
		errs = append(errs, fmt.Errorf("images.AnsibleServiceBroker cannot be empty"))
	}

	if i.WebConsole == "" {
		errs = append(errs, fmt.Errorf("images.WebConsole cannot be empty"))
	}

	if i.Console == "" {
		errs = append(errs, fmt.Errorf("images.Console cannot be empty"))
	}

	if i.EtcdBackup == "" {
		errs = append(errs, fmt.Errorf("images.EtcdBackup cannot be empty"))
	}

	if len(i.GenevaImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("images.GenevaImagePullSecret cannot be empty"))
	}

	if i.GenevaLogging == "" {
		errs = append(errs, fmt.Errorf("images.GenevaLogging cannot be empty"))
	}

	if i.GenevaTDAgent == "" {
		errs = append(errs, fmt.Errorf("images.GenevaTDAgent cannot be empty"))
	}

	if i.GenevaStatsd == "" {
		errs = append(errs, fmt.Errorf("images.GenevaStatsd cannot be empty"))
	}

	if i.MetricsBridge == "" {
		errs = append(errs, fmt.Errorf("images.MetricsBridge cannot be empty"))
	}

	if len(i.ImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("images.ImagePullSecret cannot be empty"))
	}

	return
}

func (v *PluginAPIValidator) validateCertificateConfig(c *pluginapi.CertificateConfig) (errs []error) {
	if c == nil {
		errs = append(errs, fmt.Errorf("certificateConfig cannot be nil"))
		return
	}

	if c.GenevaLogging.Key == nil {
		errs = append(errs, fmt.Errorf("GenevaLogging key cannot be empty"))
	} else if err := c.GenevaLogging.Key.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid GenevaLogging key: %v", err))
	}

	if c.GenevaLogging.Cert == nil {
		errs = append(errs, fmt.Errorf("GenevaLogging certificate cannot be empty"))
	}

	if c.GenevaMetrics.Key == nil {
		errs = append(errs, fmt.Errorf("GenevaMetrics key cannot be empty"))
	} else if err := c.GenevaMetrics.Key.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid GenevaMetrics key: %v", err))
	}

	if c.GenevaMetrics.Cert == nil {
		errs = append(errs, fmt.Errorf("GenevaMetrics certificate cannot be empty"))
	}

	return
}
