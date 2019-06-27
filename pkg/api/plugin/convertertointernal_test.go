package plugin

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/populate"
	"github.com/openshift/openshift-azure/test/util/tls"
)

func externalPluginConfig() *Config {
	// use populate.Walk to generate a fully populated
	// Config
	pc := Config{}
	populate.Walk(&pc, func(v reflect.Value) {})
	return &pc
}

func internalPluginConfig() api.Config {
	// this is the expected internal equivalent to
	// internal API Config
	return api.Config{
		SecurityPatchPackages: []string{"SecurityPatchPackages[0]"},
		PluginVersion:         "Versions.key",
		ComponentLogLevel: api.ComponentLogLevel{
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
		Certificates: api.CertificateConfig{
			GenevaLogging: api.CertKeyPair{
				Cert: tls.DummyCertificate,
				Key:  tls.DummyPrivateKey,
			},
			GenevaMetrics: api.CertKeyPair{
				Cert: tls.DummyCertificate,
				Key:  tls.DummyPrivateKey,
			},
		},
		// Container images configuration
		Images: api.ImageConfig{
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

func TestToInternal(t *testing.T) {
	// prepare external type
	var external Config
	populate.Walk(&external, func(v reflect.Value) {})
	external.PluginVersion = "should not be copied"
	// prepare internal type
	internal := internalPluginConfig()
	output, _ := ToInternal(&external, &api.Config{PluginVersion: "Versions.key"}, true)
	if !reflect.DeepEqual(*output, internal) {
		t.Errorf("unexpected diff %s", deep.Equal(*output, internal))
	}
}
