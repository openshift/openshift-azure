package plugin

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/cmp"
	"github.com/openshift/openshift-azure/pkg/util/tls"
	"github.com/openshift/openshift-azure/test/util/populate"
	testtls "github.com/openshift/openshift-azure/test/util/tls"
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
				Cert: testtls.DummyCertificate,
				Key:  testtls.DummyPrivateKey,
			},
			GenevaMetrics: api.CertKeyPair{
				Cert: testtls.DummyCertificate,
				Key:  testtls.DummyPrivateKey,
			},
			PackageRepository: api.CertKeyPair{
				Cert: testtls.DummyCertificate,
				Key:  testtls.DummyPrivateKey,
			},
		},
		// Container images configuration
		Images: api.ImageConfig{
			ImagePullSecret:           []byte("ImagePullSecret"),
			GenevaImagePullSecret:     []byte("GenevaImagePullSecret"),
			Format:                    "Versions.key.Images.Format",
			ClusterMonitoringOperator: "Versions.key.Images.ClusterMonitoringOperator",
			AzureControllers:          "Versions.key.Images.AzureControllers",
			AroAdmissionController:    "Versions.key.Images.AroAdmissionController",
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
			LogAnalyticsAgent:         "Versions.key.Images.LogAnalyticsAgent",
			MetricsServer:             "Versions.key.Images.MetricsServer",
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
		t.Errorf("unexpected diff %s", cmp.Diff(*output, internal))
	}
}

func TestToInternalSecretUpdate(t *testing.T) {
	// prepare external type
	var external Config
	populate.Walk(&external, func(v reflect.Value) {})
	external.PluginVersion = "should not be copied"

	cn := "dummy-test-certificate.local"
	key, cert, err := tls.NewCert(&tls.CertParams{
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{cn},
		},
		DNSNames:    []string{cn},
		IPAddresses: []net.IP{net.ParseIP("192.168.0.1")},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth | x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = key.Validate()
	if err != nil {
		t.Error(err)
	}

	// this test is to ensure that the following secrets can be force updated using
	// the setVersionFields flag.
	external.Certificates.GenevaLogging.Key = key
	external.Certificates.GenevaLogging.Cert = cert
	external.Certificates.GenevaMetrics.Key = key
	external.Certificates.GenevaMetrics.Cert = cert
	external.Certificates.PackageRepository.Key = key
	external.Certificates.PackageRepository.Cert = cert
	external.GenevaImagePullSecret = []byte("hello")
	external.ImagePullSecret = []byte("hello")

	old := internalPluginConfig()
	output, _ := ToInternal(&external, &old, true)
	if !reflect.DeepEqual(output.Certificates.GenevaLogging.Cert, external.Certificates.GenevaLogging.Cert) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Certificates.GenevaLogging.Cert, external.Certificates.GenevaLogging.Cert))
	}
	if !reflect.DeepEqual(output.Certificates.GenevaLogging.Key, external.Certificates.GenevaLogging.Key) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Certificates.GenevaLogging.Key, external.Certificates.GenevaLogging.Key))
	}
	if !reflect.DeepEqual(output.Certificates.GenevaMetrics.Cert, external.Certificates.GenevaMetrics.Cert) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Certificates.GenevaMetrics.Cert, external.Certificates.GenevaMetrics.Cert))
	}
	if !reflect.DeepEqual(output.Certificates.GenevaMetrics.Key, external.Certificates.GenevaMetrics.Key) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Certificates.GenevaMetrics.Key, external.Certificates.GenevaMetrics.Key))
	}
	if !reflect.DeepEqual(output.Certificates.PackageRepository.Cert, external.Certificates.PackageRepository.Cert) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Certificates.PackageRepository.Cert, external.Certificates.PackageRepository.Cert))
	}
	if !reflect.DeepEqual(output.Certificates.PackageRepository.Key, external.Certificates.PackageRepository.Key) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Certificates.PackageRepository.Key, external.Certificates.PackageRepository.Key))
	}
	if !reflect.DeepEqual(output.Images.ImagePullSecret, external.ImagePullSecret) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Images.ImagePullSecret, external.ImagePullSecret))
	}
	if !reflect.DeepEqual(output.Images.GenevaImagePullSecret, external.GenevaImagePullSecret) {
		t.Errorf("unexpected diff %s", cmp.Diff(output.Images.GenevaImagePullSecret, external.GenevaImagePullSecret))
	}
}
