package validate

import (
	"errors"
	"reflect"
	"testing"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

func TestPluginTemplateValidate(t *testing.T) {
	expectedErrs :=
		[]error{errors.New(`imageOffer should be osa`),
			errors.New(`imagePublisher should be redhat`),
			errors.New(`invalid ImageSKU ""`),
			errors.New(`invalid ClusterVersion ""`),
			errors.New(`invalid ImageVersion ""`),
			errors.New(`genevaLoggingSector cannot be empty`),
			errors.New(`genevaLoggingAccount cannot be empty`),
			errors.New(`genevaLoggingNamespace cannot be empty`),
			errors.New(`genevaLoggingControlPlaneAccount cannot be empty`),
			errors.New(`genevaMetricsAccount cannot be empty`),
			errors.New(`genevaMetricsEndpoint cannot be empty`),
			errors.New(`GenevaLogging key cannot be empty`),
			errors.New(`GenevaLogging certificate cannot be empty`),
			errors.New(`GenevaMetrics key cannot be empty`),
			errors.New(`GenevaMetrics certificate cannot be empty`),
			errors.New(`images.Format cannot be empty`),
			errors.New(`images.GenevaImagePullSecret cannot be empty`),
			errors.New(`images.ClusterMonitoringOperator cannot be empty`),
			errors.New(`images.AzureControllers cannot be empty`),
			errors.New(`images.PrometheusOperatorBase cannot be empty`),
			errors.New(`images.PrometheusBase cannot be empty`),
			errors.New(`images.PrometheusConfigReloaderBase cannot be empty`),
			errors.New(`images.ConfigReloaderBase cannot be empty`),
			errors.New(`images.AlertManagerBase cannot be empty`),
			errors.New(`images.NodeExporterBase cannot be empty`),
			errors.New(`images.GrafanaBase cannot be empty`),
			errors.New(`images.KubeStateMetricsBase cannot be empty`),
			errors.New(`images.KubeRbacProxyBase cannot be empty`),
			errors.New(`images.OAuthProxyBase cannot be empty`),
			errors.New(`images.MasterEtcd cannot be empty`),
			errors.New(`images.ControlPlane cannot be empty`),
			errors.New(`images.Node cannot be empty`),
			errors.New(`images.ServiceCatalog cannot be empty`),
			errors.New(`images.Sync cannot be empty`),
			errors.New(`images.Startup cannot be empty`),
			errors.New(`images.TemplateServiceBroker cannot be empty`),
			errors.New(`images.Registry cannot be empty`),
			errors.New(`images.Router cannot be empty`),
			errors.New(`images.RegistryConsole cannot be empty`),
			errors.New(`images.AnsibleServiceBroker cannot be empty`),
			errors.New(`images.WebConsole cannot be empty`),
			errors.New(`images.Console cannot be empty`),
			errors.New(`images.EtcdBackup cannot be empty`),
		}

	template := pluginapi.Config{}
	v := PluginAPIValidator{}
	errs := v.Validate(&template)
	if !reflect.DeepEqual(errs, expectedErrs) {
		t.Errorf("expected errors:")
		for _, err := range expectedErrs {
			t.Errorf("\t%v", err)
		}
		t.Error("received errors:")
		for _, err := range errs {
			t.Errorf("\t%v", err)
		}
	}
}
