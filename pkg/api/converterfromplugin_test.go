package api

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
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
		ComponentLogLevel: ComponentLogLevel{
			APIServer:         to.IntPtr(1),
			ControllerManager: to.IntPtr(1),
			Node:              to.IntPtr(1),
		},
		// generic Offering configuration
		ImageOffer:               "Versions.key.ImageOffer",
		ImagePublisher:           "Versions.key.ImagePublisher",
		ImageSKU:                 "Versions.key.ImageSKU",
		ImageVersion:             "Versions.key.ImageVersion",
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
			ImagePullSecret:           []byte("ImagePullSecret"),
			GenevaImagePullSecret:     []byte("GenevaImagePullSecret"),
			Format:                    "Versions.key.Images.Format",
			ClusterMonitoringOperator: "Versions.key.Images.ClusterMonitoringOperator",
			AzureControllers:          "Versions.key.Images.AzureControllers",
			Canary:                    "Versions.key.Images.Canary",
			PrometheusOperator:        "Versions.key.Images.PrometheusOperator",
			Prometheus:                "Versions.key.Images.Prometheus",
			PrometheusConfigReloader:  "Versions.key.Images.PrometheusConfigReloader",
			ConfigReloader:            "Versions.key.Images.ConfigReloader",
			AlertManager:              "Versions.key.Images.AlertManager",
			NodeExporter:              "Versions.key.Images.NodeExporter",
			Grafana:                   "Versions.key.Images.Grafana",
			KubeStateMetrics:          "Versions.key.Images.KubeStateMetrics",
			KubeRbacProxy:             "Versions.key.Images.KubeRbacProxy",
			OAuthProxy:                "Versions.key.Images.OAuthProxy",
			MasterEtcd:                "Versions.key.Images.MasterEtcd",
			ControlPlane:              "Versions.key.Images.ControlPlane",
			Node:                      "Versions.key.Images.Node",
			ServiceCatalog:            "Versions.key.Images.ServiceCatalog",
			Sync:                      "Versions.key.Images.Sync",
			Startup:                   "Versions.key.Images.Startup",
			TemplateServiceBroker:     "Versions.key.Images.TemplateServiceBroker",
			TLSProxy:                  "Versions.key.Images.TLSProxy",
			Registry:                  "Versions.key.Images.Registry",
			Router:                    "Versions.key.Images.Router",
			RegistryConsole:           "Versions.key.Images.RegistryConsole",
			AnsibleServiceBroker:      "Versions.key.Images.AnsibleServiceBroker",
			WebConsole:                "Versions.key.Images.WebConsole",
			Console:                   "Versions.key.Images.Console",
			EtcdBackup:                "Versions.key.Images.EtcdBackup",
			Httpd:                     "Versions.key.Images.Httpd",
			GenevaLogging:             "Versions.key.Images.GenevaLogging",
			GenevaTDAgent:             "Versions.key.Images.GenevaTDAgent",
			GenevaStatsd:              "Versions.key.Images.GenevaStatsd",
			MetricsBridge:             "Versions.key.Images.MetricsBridge",
		},
	}
}

func TestConvertFromPlugin(t *testing.T) {
	// prepare external type
	var external plugin.Config
	populate.Walk(&external, func(v reflect.Value) {})
	// prepare internal type
	internal := internalPluginConfig()
	output, _ := ConvertFromPlugin(&external, &internal, "Versions.key")
	if !reflect.DeepEqual(*output, internal) {
		t.Errorf("unexpected diff %s", deep.Equal(*output, internal))
	}
}
