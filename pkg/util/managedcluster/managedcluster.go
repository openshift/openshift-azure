package managedcluster

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

// ReadConfig returns a config object from a config file
func ReadConfig(path string) (*api.OpenShiftManagedCluster, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, err
	}

	return cs, nil
}

// RestConfigFromV1Config takes a v1 config and returns a kubeconfig
func RestConfigFromV1Config(kc *v1.Config) (*rest.Config, error) {
	var c kapi.Config
	err := latest.Scheme.Convert(kc, &c, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(c, &clientcmd.ConfigOverrides{})
	return kubeconfig.ClientConfig()
}

// PluginTemplate returns the plugin template which was used to create the cluster specified by cs
func PluginTemplate(cs *api.OpenShiftManagedCluster) *pluginapi.Config {
	return &pluginapi.Config{
		ClusterVersion:                       cs.Config.ClusterVersion,
		ImageOffer:                           cs.Config.ImageOffer,
		ImagePublisher:                       cs.Config.ImagePublisher,
		ImageSKU:                             cs.Config.ImageSKU,
		ImageVersion:                         cs.Config.ImageVersion,
		GenevaLoggingSector:                  cs.Config.GenevaLoggingSector,
		GenevaLoggingNamespace:               cs.Config.GenevaLoggingNamespace,
		GenevaLoggingAccount:                 cs.Config.GenevaLoggingAccount,
		GenevaMetricsAccount:                 cs.Config.GenevaMetricsAccount,
		GenevaMetricsEndpoint:                cs.Config.GenevaMetricsEndpoint,
		GenevaLoggingControlPlaneAccount:     cs.Config.GenevaLoggingControlPlaneAccount,
		GenevaLoggingControlPlaneEnvironment: cs.Config.GenevaLoggingControlPlaneEnvironment,
		GenevaLoggingControlPlaneRegion:      cs.Config.GenevaLoggingControlPlaneRegion,
		Certificates: pluginapi.CertificateConfig{
			GenevaLogging: pluginapi.CertKeyPair{
				Key:  cs.Config.Certificates.GenevaLogging.Key,
				Cert: cs.Config.Certificates.GenevaLogging.Cert,
			},
			GenevaMetrics: pluginapi.CertKeyPair{
				Key:  cs.Config.Certificates.GenevaMetrics.Key,
				Cert: cs.Config.Certificates.GenevaMetrics.Cert,
			},
		},
		Images: pluginapi.ImageConfig{
			ImagePullSecret:              cs.Config.Images.ImagePullSecret,
			GenevaImagePullSecret:        cs.Config.Images.GenevaImagePullSecret,
			Format:                       cs.Config.Images.Format,
			ClusterMonitoringOperator:    cs.Config.Images.ClusterMonitoringOperator,
			AzureControllers:             cs.Config.Images.AzureControllers,
			PrometheusOperatorBase:       cs.Config.Images.PrometheusOperatorBase,
			PrometheusBase:               cs.Config.Images.PrometheusBase,
			PrometheusConfigReloaderBase: cs.Config.Images.PrometheusConfigReloaderBase,
			ConfigReloaderBase:           cs.Config.Images.ConfigReloaderBase,
			AlertManagerBase:             cs.Config.Images.AlertManagerBase,
			NodeExporterBase:             cs.Config.Images.NodeExporterBase,
			GrafanaBase:                  cs.Config.Images.GrafanaBase,
			KubeStateMetricsBase:         cs.Config.Images.KubeStateMetricsBase,
			KubeRbacProxyBase:            cs.Config.Images.KubeRbacProxyBase,
			OAuthProxyBase:               cs.Config.Images.OAuthProxyBase,
			MasterEtcd:                   cs.Config.Images.MasterEtcd,
			ControlPlane:                 cs.Config.Images.ControlPlane,
			Node:                         cs.Config.Images.Node,
			ServiceCatalog:               cs.Config.Images.ServiceCatalog,
			Sync:                         cs.Config.Images.Sync,
			TemplateServiceBroker:        cs.Config.Images.TemplateServiceBroker,
			Registry:                     cs.Config.Images.Registry,
			Router:                       cs.Config.Images.Router,
			RegistryConsole:              cs.Config.Images.RegistryConsole,
			AnsibleServiceBroker:         cs.Config.Images.AnsibleServiceBroker,
			WebConsole:                   cs.Config.Images.WebConsole,
			Console:                      cs.Config.Images.Console,
			EtcdBackup:                   cs.Config.Images.EtcdBackup,
			GenevaLogging:                cs.Config.Images.GenevaLogging,
			GenevaTDAgent:                cs.Config.Images.GenevaTDAgent,
			GenevaStatsd:                 cs.Config.Images.GenevaStatsd,
			MetricsBridge:                cs.Config.Images.MetricsBridge,
		},
	}
}
