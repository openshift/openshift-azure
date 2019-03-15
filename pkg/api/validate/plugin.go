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

	if !rxPluginVersion.MatchString(c.PluginVersion) {
		errs = append(errs, fmt.Errorf("invalid pluginVersion %q", c.PluginVersion))
	}

	errs = append(errs, validateComponentLogLevel(&c.ComponentLogLevel)...)

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

func validateComponentLogLevel(c *pluginapi.ComponentLogLevel) (errs []error) {
	// cant't set logging level > 7 due to:
	// https://bugzilla.redhat.com/show_bug.cgi?id=1689366
	if c.APIServer < 0 || c.APIServer > 7 {
		errs = append(errs, fmt.Errorf("invalid componentLogLevel.apiServer %d", c.APIServer))
	}

	if c.ControllerManager < 0 || c.ControllerManager > 20 {
		errs = append(errs, fmt.Errorf("invalid componentLogLevel.controllerManager %d", c.ControllerManager))
	}

	if c.Node < 0 || c.Node > 20 {
		errs = append(errs, fmt.Errorf("invalid componentLogLevel.node %d", c.Node))
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

	if i.PrometheusOperator == "" {
		errs = append(errs, fmt.Errorf("invalid images.prometheusOperator %q", i.PrometheusOperator))
	}

	if i.Prometheus == "" {
		errs = append(errs, fmt.Errorf("invalid images.prometheus %q", i.Prometheus))
	}

	if i.PrometheusConfigReloader == "" {
		errs = append(errs, fmt.Errorf("invalid images.prometheusConfigReloader %q", i.PrometheusConfigReloader))
	}

	if i.ConfigReloader == "" {
		errs = append(errs, fmt.Errorf("invalid images.configReloader %q", i.ConfigReloader))
	}

	if i.AlertManager == "" {
		errs = append(errs, fmt.Errorf("invalid images.alertManager %q", i.AlertManager))
	}

	if i.NodeExporter == "" {
		errs = append(errs, fmt.Errorf("invalid images.nodeExporter %q", i.NodeExporter))
	}

	if i.Grafana == "" {
		errs = append(errs, fmt.Errorf("invalid images.grafana %q", i.Grafana))
	}

	if i.KubeStateMetrics == "" {
		errs = append(errs, fmt.Errorf("invalid images.kubeStateMetrics %q", i.KubeStateMetrics))
	}

	if i.KubeRbacProxy == "" {
		errs = append(errs, fmt.Errorf("invalid images.kubeRbacProxy %q", i.KubeRbacProxy))
	}

	if i.OAuthProxy == "" {
		errs = append(errs, fmt.Errorf("invalid images.oAuthProxy %q", i.OAuthProxy))
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

	if i.TLSProxy == "" {
		errs = append(errs, fmt.Errorf("invalid images.TLSProxy %q", i.TLSProxy))
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
