package api

import (
	plugin "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

// ConvertFromPlugin converts external plugin API config type into
// internal API Config type
func ConvertFromPlugin(in plugin.Config) Config {
	out := Config{
		// generic OSA offering configuration
		ImageOffer:     in.ImageOffer,
		ImagePublisher: in.ImagePublisher,
		ImageSKU:       in.ImageSKU,
		ImageVersion:   in.ImageVersion,
		// Geneva integration configuration
		GenevaLoggingSector:              in.GenevaLoggingSector,
		GenevaLoggingNamespace:           in.GenevaLoggingNamespace,
		GenevaLoggingAccount:             in.GenevaLoggingAccount,
		GenevaMetricsAccount:             in.GenevaMetricsAccount,
		GenevaMetricsEndpoint:            in.GenevaMetricsEndpoint,
		GenevaLoggingControlPlaneAccount: in.GenevaLoggingControlPlaneAccount,
		Certificates: CertificateConfig{
			GenevaLogging: CertKeyPair{
				Cert: in.Certificates.GenevaLogging.Cert,
				Key:  in.Certificates.GenevaLogging.Key,
			},
			GenevaMetrics: CertKeyPair{
				Cert: in.Certificates.GenevaMetrics.Cert,
				Key:  in.Certificates.GenevaMetrics.Key,
			},
		},
		// Container images configuration
		Images: ImageConfig{
			GenevaImagePullSecret:        in.Images.GenevaImagePullSecret,
			Format:                       in.Images.Format,
			ClusterMonitoringOperator:    in.Images.ClusterMonitoringOperator,
			AzureControllers:             in.Images.AzureControllers,
			PrometheusOperatorBase:       in.Images.PrometheusOperatorBase,
			PrometheusBase:               in.Images.PrometheusBase,
			PrometheusConfigReloaderBase: in.Images.PrometheusConfigReloaderBase,
			ConfigReloaderBase:           in.Images.ConfigReloaderBase,
			AlertManagerBase:             in.Images.AlertManagerBase,
			NodeExporterBase:             in.Images.NodeExporterBase,
			GrafanaBase:                  in.Images.GrafanaBase,
			KubeStateMetricsBase:         in.Images.KubeStateMetricsBase,
			KubeRbacProxyBase:            in.Images.KubeRbacProxyBase,
			OAuthProxyBase:               in.Images.OAuthProxyBase,
			MasterEtcd:                   in.Images.MasterEtcd,
			ControlPlane:                 in.Images.ControlPlane,
			Node:                         in.Images.Node,
			ServiceCatalog:               in.Images.ServiceCatalog,
			Sync:                         in.Images.Sync,
			TemplateServiceBroker:        in.Images.TemplateServiceBroker,
			Registry:                     in.Images.Registry,
			Router:                       in.Images.Router,
			RegistryConsole:              in.Images.RegistryConsole,
			AnsibleServiceBroker:         in.Images.AnsibleServiceBroker,
			WebConsole:                   in.Images.WebConsole,
			Console:                      in.Images.Console,
			EtcdBackup:                   in.Images.EtcdBackup,
			GenevaLogging:                in.Images.GenevaLogging,
			GenevaTDAgent:                in.Images.GenevaTDAgent,
			GenevaStatsd:                 in.Images.GenevaStatsd,
			MetricsBridge:                in.Images.MetricsBridge,
		},
	}
	return out
}
