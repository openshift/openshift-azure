package api

import (
	"reflect"
	"testing"

	"github.com/go-test/deep"

	plugin "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/test/util/populate"
	"github.com/openshift/openshift-azure/test/util/tls"
)

func externalPluginConfig() *plugin.Config {
	// use populate.Walk to generate a fully populated
	// plugin.Config
	pc := plugin.Config{}
	populate.Walk(&pc, func(v reflect.Value) {})
	return &pc
}

func internalPluginConfig() Config {
	// this is the expected internal equivalent to
	// internal API Config
	return Config{

		PluginVersion: "PluginVersion",
		// generic Offering configuration
		ImageOffer:               "ImageOffer",
		ImagePublisher:           "ImagePublisher",
		ImageSKU:                 "ImageSKU",
		ImageVersion:             "ImageVersion",
		SSHSourceAddressPrefixes: []string{"SSHSourceAddressPrefixes[0]"},
		// Geneva intergration configuration
		GenevaLoggingSector:                  "GenevaLoggingSector",
		GenevaLoggingNamespace:               "GenevaLoggingNamespace",
		GenevaLoggingAccount:                 "GenevaLoggingAccount",
		GenevaMetricsAccount:                 "GenevaMetricsAccount",
		GenevaMetricsEndpoint:                "GenevaMetricsEndpoint",
		GenevaLoggingControlPlaneAccount:     "GenevaLoggingControlPlaneAccount",
		GenevaLoggingControlPlaneEnvironment: "GenevaLoggingControlPlaneEnvironment",
		GenevaLoggingControlPlaneRegion:      "GenevaLoggingControlPlaneRegion",
		Certificates: CertificateConfig{
			GenevaLogging: CertKeyPair{
				Cert: tls.GetDummyCertificate(),
				Key:  tls.GetDummyPrivateKey(),
			},
			GenevaMetrics: CertKeyPair{
				Cert: tls.GetDummyCertificate(),
				Key:  tls.GetDummyPrivateKey(),
			},
		},
		// Container images configuration
		Images: ImageConfig{
			ImagePullSecret:           []byte("Images.ImagePullSecret"),
			GenevaImagePullSecret:     []byte("Images.GenevaImagePullSecret"),
			Format:                    "Images.Format",
			ClusterMonitoringOperator: "Images.ClusterMonitoringOperator",
			AzureControllers:          "Images.AzureControllers",
			PrometheusOperator:        "Images.PrometheusOperator",
			Prometheus:                "Images.Prometheus",
			PrometheusConfigReloader:  "Images.PrometheusConfigReloader",
			ConfigReloader:            "Images.ConfigReloader",
			AlertManager:              "Images.AlertManager",
			NodeExporter:              "Images.NodeExporter",
			Grafana:                   "Images.Grafana",
			KubeStateMetrics:          "Images.KubeStateMetrics",
			KubeRbacProxy:             "Images.KubeRbacProxy",
			OAuthProxy:                "Images.OAuthProxy",
			MasterEtcd:                "Images.MasterEtcd",
			ControlPlane:              "Images.ControlPlane",
			Node:                      "Images.Node",
			ServiceCatalog:            "Images.ServiceCatalog",
			Sync:                      "Images.Sync",
			Startup:                   "Images.Startup",
			TemplateServiceBroker:     "Images.TemplateServiceBroker",
			Registry:                  "Images.Registry",
			Router:                    "Images.Router",
			RegistryConsole:           "Images.RegistryConsole",
			AnsibleServiceBroker:      "Images.AnsibleServiceBroker",
			WebConsole:                "Images.WebConsole",
			Console:                   "Images.Console",
			EtcdBackup:                "Images.EtcdBackup",
			GenevaLogging:             "Images.GenevaLogging",
			GenevaTDAgent:             "Images.GenevaTDAgent",
			GenevaStatsd:              "Images.GenevaStatsd",
			MetricsBridge:             "Images.MetricsBridge",
		},
	}
}

func TestConvertFromPlugin(t *testing.T) {
	// prepare external type
	var external plugin.Config
	populate.Walk(&external, func(v reflect.Value) {})
	// prepare internal type
	internal := internalPluginConfig()
	output := ConvertFromPlugin(&external, &internal)
	if !reflect.DeepEqual(*output, internal) {
		t.Errorf("unexpected diff %s", deep.Equal(output, internal))
	}
}
