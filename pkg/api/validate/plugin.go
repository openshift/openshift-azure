package validate

import (
	"fmt"
	"net"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
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

	for _, prefix := range c.SSHSourceAddressPrefixes {
		if _, _, err := net.ParseCIDR(prefix); err != nil {
			errs = append(errs, fmt.Errorf("invalid sshSourceAddressPrefix %q", prefix))
		}
	}

	for version, versionConfig := range c.Versions {
		if !rxPluginVersion.MatchString(version) {
			errs = append(errs, fmt.Errorf("invalid versions[%q]", version))
		}

		errs = append(errs, validateVersionConfig(fmt.Sprintf("versions[%q]", version), version, &versionConfig)...)
	}
	if _, found := c.Versions[c.PluginVersion]; !found {
		errs = append(errs, fmt.Errorf("missing versions key %q", c.PluginVersion))
	}

	errs = append(errs, validateCertificateConfig(&c.Certificates)...)

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

	if len(c.GenevaImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("invalid genevaImagePullSecret %q", c.GenevaImagePullSecret))
	}

	if len(c.ImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("invalid imagePullSecret %q", c.ImagePullSecret))
	}

	return
}

func validateComponentLogLevel(c *pluginapi.ComponentLogLevel) (errs []error) {
	// can't set logging level > 7 due to:
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

func validateVersionConfig(path string, version string, vc *pluginapi.VersionConfig) (errs []error) {
	if vc.ImageOffer != "osa" {
		errs = append(errs, fmt.Errorf("invalid %s.imageOffer %q", path, vc.ImageOffer))
	}

	if vc.ImagePublisher != "redhat" {
		errs = append(errs, fmt.Errorf("invalid %s.imagePublisher %q", path, vc.ImagePublisher))
	}

	switch vc.ImageSKU {
	case "osa_311":
	default:
		errs = append(errs, fmt.Errorf("invalid %s.imageSKU %q", path, vc.ImageSKU))
	}

	if !rxImageVersion.MatchString(vc.ImageVersion) {
		errs = append(errs, fmt.Errorf("invalid %s.imageVersion %q", path, vc.ImageVersion))
	}

	errs = append(errs, validateImageConfig(fmt.Sprintf("%s.images", path), version, &vc.Images)...)

	return
}

func validateImageConfig(path, version string, i *pluginapi.ImageConfig) (errs []error) {
	if i == nil {
		errs = append(errs, fmt.Errorf("imageConfig cannot be nil"))
		return
	}

	if i.AlertManager == "" {
		errs = append(errs, fmt.Errorf("invalid %s.alertManager %q", path, i.AlertManager))
	}

	if i.AnsibleServiceBroker == "" {
		errs = append(errs, fmt.Errorf("invalid %s.ansibleServiceBroker %q", path, i.AnsibleServiceBroker))
	}

	if i.ClusterMonitoringOperator == "" {
		errs = append(errs, fmt.Errorf("invalid %s.clusterMonitoringOperator %q", path, i.ClusterMonitoringOperator))
	}

	if i.ConfigReloader == "" {
		errs = append(errs, fmt.Errorf("invalid %s.configReloader %q", path, i.ConfigReloader))
	}

	if i.Console == "" {
		errs = append(errs, fmt.Errorf("invalid %s.console %q", path, i.Console))
	}

	if i.ControlPlane == "" {
		errs = append(errs, fmt.Errorf("invalid %s.controlPlane %q", path, i.ControlPlane))
	}

	if i.Grafana == "" {
		errs = append(errs, fmt.Errorf("invalid %s.grafana %q", path, i.Grafana))
	}

	if i.KubeRbacProxy == "" {
		errs = append(errs, fmt.Errorf("invalid %s.kubeRbacProxy %q", path, i.KubeRbacProxy))
	}

	if i.KubeStateMetrics == "" {
		errs = append(errs, fmt.Errorf("invalid %s.kubeStateMetrics %q", path, i.KubeStateMetrics))
	}

	if i.Node == "" {
		errs = append(errs, fmt.Errorf("invalid %s.node %q", path, i.Node))
	}

	if i.NodeExporter == "" {
		errs = append(errs, fmt.Errorf("invalid %s.nodeExporter %q", path, i.NodeExporter))
	}

	if i.OAuthProxy == "" {
		errs = append(errs, fmt.Errorf("invalid %s.oAuthProxy %q", path, i.OAuthProxy))
	}

	if i.Prometheus == "" {
		errs = append(errs, fmt.Errorf("invalid %s.prometheus %q", path, i.Prometheus))
	}

	if i.PrometheusConfigReloader == "" {
		errs = append(errs, fmt.Errorf("invalid %s.prometheusConfigReloader %q", path, i.PrometheusConfigReloader))
	}

	if i.PrometheusOperator == "" {
		errs = append(errs, fmt.Errorf("invalid %s.prometheusOperator %q", path, i.PrometheusOperator))
	}

	if i.Registry == "" {
		errs = append(errs, fmt.Errorf("invalid %s.registry %q", path, i.Registry))
	}

	if i.RegistryConsole == "" {
		errs = append(errs, fmt.Errorf("invalid %s.registryConsole %q", path, i.RegistryConsole))
	}

	if i.Router == "" {
		errs = append(errs, fmt.Errorf("invalid %s.router %q", path, i.Router))
	}

	if i.ServiceCatalog == "" {
		errs = append(errs, fmt.Errorf("invalid %s.serviceCatalog %q", path, i.ServiceCatalog))
	}

	if i.TemplateServiceBroker == "" {
		errs = append(errs, fmt.Errorf("invalid %s.templateServiceBroker %q", path, i.TemplateServiceBroker))
	}

	if i.WebConsole == "" {
		errs = append(errs, fmt.Errorf("invalid %s.webConsole %q", path, i.WebConsole))
	}

	if i.Format == "" {
		errs = append(errs, fmt.Errorf("invalid %s.format %q", path, i.Format))
	}

	if i.Httpd == "" {
		errs = append(errs, fmt.Errorf("invalid %s.httpd %q", path, i.Httpd))
	}

	if i.MasterEtcd == "" {
		errs = append(errs, fmt.Errorf("invalid %s.masterEtcd %q", path, i.MasterEtcd))
	}

	if i.GenevaLogging == "" {
		errs = append(errs, fmt.Errorf("invalid %s.genevaLogging %q", path, i.GenevaLogging))
	}

	if i.GenevaStatsd == "" {
		errs = append(errs, fmt.Errorf("invalid %s.genevaStatsd %q", path, i.GenevaStatsd))
	}

	if i.GenevaTDAgent == "" {
		errs = append(errs, fmt.Errorf("invalid %s.genevaTDAgent %q", path, i.GenevaTDAgent))
	}

	if i.AzureControllers == "" {
		errs = append(errs, fmt.Errorf("invalid %s.azureControllers %q", path, i.AzureControllers))
	}

	if (i.Canary == "") != (version == "v3.2") {
		errs = append(errs, fmt.Errorf("invalid %s.canary %q", path, i.Canary))
	}

	if i.EtcdBackup == "" {
		errs = append(errs, fmt.Errorf("invalid %s.etcdBackup %q", path, i.EtcdBackup))
	}

	if i.MetricsBridge == "" {
		errs = append(errs, fmt.Errorf("invalid %s.metricsBridge %q", path, i.MetricsBridge))
	}

	if i.Startup == "" {
		errs = append(errs, fmt.Errorf("invalid %s.startup %q", path, i.Startup))
	}

	if i.Sync == "" {
		errs = append(errs, fmt.Errorf("invalid %s.sync %q", path, i.Sync))
	}

	if i.TLSProxy == "" {
		errs = append(errs, fmt.Errorf("invalid %s.tlsProxy %q", path, i.TLSProxy))
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
