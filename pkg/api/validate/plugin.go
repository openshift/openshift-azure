package validate

import (
	"fmt"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

// Validate validates an Plugin API external template/config struct
func (v *PluginAPIValidator) Validate(t *pluginapi.Config) (errs []error) {
	if t.ImageOffer != "osa" {
		errs = append(errs, fmt.Errorf("imageOffer should be osa"))
	}
	if t.ImagePublisher != "redhat" {
		errs = append(errs, fmt.Errorf("imagePublisher should be redhat"))
	}
	// validate valid ImageSKU's
	switch t.ImageSKU {
	case "osa_311":
	default:
		errs = append(errs, fmt.Errorf("invalid ImageSKU %q", t.ImageSKU))
	}
	// validate ClusterVersion
	if !clusterVersion.MatchString(t.ClusterVersion) {
		errs = append(errs, fmt.Errorf("invalid ClusterVersion %q", t.ClusterVersion))
	}
	// validate ComponentLogLevel
	errs = append(errs, v.validatePluginTemplateComponentLogLevels(&t.ComponentLogLevel)...)
	// validate ImageVersion
	if !imageVersion.MatchString(t.ImageVersion) {
		errs = append(errs, fmt.Errorf("invalid ImageVersion %q", t.ImageVersion))
	}
	// validate geneva configuration inputs
	if t.GenevaLoggingSector == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingSector cannot be empty"))
	}
	if t.GenevaLoggingAccount == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingAccount cannot be empty"))
	}
	if t.GenevaLoggingNamespace == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingNamespace cannot be empty"))
	}
	if t.GenevaLoggingControlPlaneAccount == "" {
		errs = append(errs, fmt.Errorf("genevaLoggingControlPlaneAccount cannot be empty"))
	}
	if t.GenevaMetricsAccount == "" {
		errs = append(errs, fmt.Errorf("genevaMetricsAccount cannot be empty"))
	}
	if t.GenevaMetricsEndpoint == "" {
		errs = append(errs, fmt.Errorf("genevaMetricsEndpoint cannot be empty"))
	}
	// validate certificates
	errs = append(errs, v.validatePluginTemplateCertificates(&t.Certificates)...)
	errs = append(errs, v.validatePluginTemplateImages(&t.Images)...)
	return errs
}

func (v *PluginAPIValidator) validatePluginTemplateComponentLogLevels(c *pluginapi.ComponentLogLevel) (errs []error) {
	if c.APIServer < 0 || c.APIServer > 20 {
		errs = append(errs, fmt.Errorf("invalid componentLogLevel.apiServer %d", c.APIServer))
	}
	if c.ControllerManager < 0 || c.ControllerManager > 20 {
		errs = append(errs, fmt.Errorf("invalid componentLogLevel.controllerManager %d", c.ControllerManager))
	}
	if c.Node < 0 || c.Node > 20 {
		errs = append(errs, fmt.Errorf("invalid componentLogLevel.node %d", c.Node))
	}
	return errs
}

func (v *PluginAPIValidator) validatePluginTemplateCertificates(c *pluginapi.CertificateConfig) (errs []error) {
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
	return errs
}

func (v *PluginAPIValidator) validatePluginTemplateImages(i *pluginapi.ImageConfig) (errs []error) {
	if i.Format == "" {
		errs = append(errs, fmt.Errorf("images.Format cannot be empty"))
	}
	if len(i.GenevaImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("images.GenevaImagePullSecret cannot be empty"))
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

	return errs
}
