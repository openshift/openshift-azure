package api

import (
	plugin "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

func ConvertFromPlugin(in plugin.Config) Config {
	out := Config{}

	if in.ImageOffer != "" {
		out.ImageOffer = in.ImageOffer
	}
	if in.ImagePublisher != "" {
		out.ImagePublisher = in.ImagePublisher
	}
	if in.ImageSKU != "" {
		out.ImageSKU = in.ImageSKU
	}
	if in.ImageVersion != "" {
		out.ImageVersion = in.ImageVersion
	}

	mergePluginCertificateConfig(in.Certificates, &out.Certificates)
	mergePluginImageConfig(in.Images, &out.Images)

	if in.GenevaLoggingSector != "" {
		out.GenevaLoggingSector = in.GenevaLoggingSector
	}
	if in.GenevaLoggingNamespace != "" {
		out.GenevaLoggingNamespace = in.GenevaLoggingNamespace
	}
	if in.GenevaLoggingAccount != "" {
		out.GenevaLoggingAccount = in.GenevaLoggingAccount
	}
	if in.GenevaMetricsAccount != "" {
		out.GenevaMetricsAccount = in.GenevaMetricsAccount
	}
	if in.GenevaMetricsEndpoint != "" {
		out.GenevaMetricsEndpoint = in.GenevaMetricsEndpoint
	}
	if in.GenevaLoggingControlPlaneAccount != "" {
		out.GenevaLoggingControlPlaneAccount = in.GenevaLoggingControlPlaneAccount
	}
	return out
}

func mergePluginCertificateConfig(in plugin.CertificateConfig, out *CertificateConfig) {
	mergePluginCertKeyPair(in.GenevaLogging, &out.GenevaLogging)
	mergePluginCertKeyPair(in.GenevaMetrics, &out.GenevaMetrics)
	return
}

func mergePluginCertKeyPair(in plugin.CertKeyPair, out *CertKeyPair) {
	if in.Cert != nil {
		out.Cert = in.Cert
		out.Key = in.Key
	}
	return
}

func mergePluginImageConfig(in plugin.ImageConfig, out *ImageConfig) {
	if len(in.GenevaImagePullSecret) > 0 {
		out.GenevaImagePullSecret = in.GenevaImagePullSecret
	}
	if in.Format != "" {
		out.Format = in.Format
	}
	if in.ClusterMonitoringOperator != "" {
		out.ClusterMonitoringOperator = in.ClusterMonitoringOperator
	}
	if in.AzureControllers != "" {
		out.AzureControllers = in.AzureControllers
	}
	if in.PrometheusOperatorBase != "" {
		out.PrometheusOperatorBase = in.PrometheusOperatorBase
	}
	if in.PrometheusBase != "" {
		out.PrometheusBase = in.PrometheusBase
	}
	if in.PrometheusConfigReloaderBase != "" {
		out.PrometheusConfigReloaderBase = in.PrometheusConfigReloaderBase
	}
	if in.ConfigReloaderBase != "" {
		out.ConfigReloaderBase = in.ConfigReloaderBase
	}
	if in.AlertManagerBase != "" {
		out.AlertManagerBase = in.AlertManagerBase
	}
	if in.NodeExporterBase != "" {
		out.NodeExporterBase = in.NodeExporterBase
	}
	if in.GrafanaBase != "" {
		out.GrafanaBase = in.GrafanaBase
	}
	if in.KubeStateMetricsBase != "" {
		out.KubeStateMetricsBase = in.KubeStateMetricsBase
	}
	if in.KubeRbacProxyBase != "" {
		out.KubeRbacProxyBase = in.KubeRbacProxyBase
	}
	if in.OAuthProxyBase != "" {
		out.OAuthProxyBase = in.OAuthProxyBase
	}
	if in.MasterEtcd != "" {
		out.MasterEtcd = in.MasterEtcd
	}
	if in.ControlPlane != "" {
		out.ControlPlane = in.ControlPlane
	}
	if in.Node != "" {
		out.Node = in.Node
	}
	if in.ServiceCatalog != "" {
		out.ServiceCatalog = in.ServiceCatalog
	}
	if in.Sync != "" {
		out.Sync = in.Sync
	}
	if in.TemplateServiceBroker != "" {
		out.TemplateServiceBroker = in.TemplateServiceBroker
	}
	if in.Registry != "" {
		out.Registry = in.Registry
	}
	if in.Router != "" {
		out.Router = in.Router
	}
	if in.RegistryConsole != "" {
		out.RegistryConsole = in.RegistryConsole
	}
	if in.AnsibleServiceBroker != "" {
		out.AnsibleServiceBroker = in.AnsibleServiceBroker
	}
	if in.WebConsole != "" {
		out.WebConsole = in.WebConsole
	}
	if in.Console != "" {
		out.Console = in.Console
	}
	if in.EtcdBackup != "" {
		out.EtcdBackup = in.EtcdBackup
	}
	if in.GenevaLogging != "" {
		out.GenevaLogging = in.GenevaLogging
	}
	if in.GenevaTDAgent != "" {
		out.GenevaTDAgent = in.GenevaTDAgent
	}
	if in.GenevaStatsd != "" {
		out.GenevaStatsd = in.GenevaStatsd
	}
	if in.MetricsBridge != "" {
		out.MetricsBridge = in.MetricsBridge
	}
	return
}
