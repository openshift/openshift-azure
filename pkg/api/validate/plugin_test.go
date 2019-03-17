package validate

import (
	"errors"
	"reflect"
	"testing"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

func TestPluginTemplateValidate(t *testing.T) {
	expectedErrs :=
		[]error{
			errors.New(`invalid pluginVersion ""`),
			errors.New(`invalid versions[""]`),
			errors.New(`invalid versions[""].imageOffer ""`),
			errors.New(`invalid versions[""].imagePublisher ""`),
			errors.New(`invalid versions[""].imageSKU ""`),
			errors.New(`invalid versions[""].imageVersion ""`),
			errors.New(`invalid versions[""].images.alertManager ""`),
			errors.New(`invalid versions[""].images.ansibleServiceBroker ""`),
			errors.New(`invalid versions[""].images.clusterMonitoringOperator ""`),
			errors.New(`invalid versions[""].images.configReloader ""`),
			errors.New(`invalid versions[""].images.console ""`),
			errors.New(`invalid versions[""].images.controlPlane ""`),
			errors.New(`invalid versions[""].images.grafana ""`),
			errors.New(`invalid versions[""].images.kubeRbacProxy ""`),
			errors.New(`invalid versions[""].images.kubeStateMetrics ""`),
			errors.New(`invalid versions[""].images.node ""`),
			errors.New(`invalid versions[""].images.nodeExporter ""`),
			errors.New(`invalid versions[""].images.oAuthProxy ""`),
			errors.New(`invalid versions[""].images.prometheus ""`),
			errors.New(`invalid versions[""].images.prometheusConfigReloader ""`),
			errors.New(`invalid versions[""].images.prometheusOperator ""`),
			errors.New(`invalid versions[""].images.registry ""`),
			errors.New(`invalid versions[""].images.registryConsole ""`),
			errors.New(`invalid versions[""].images.router ""`),
			errors.New(`invalid versions[""].images.serviceCatalog ""`),
			errors.New(`invalid versions[""].images.templateServiceBroker ""`),
			errors.New(`invalid versions[""].images.webConsole ""`),
			errors.New(`invalid versions[""].images.format ""`),
			errors.New(`invalid versions[""].images.httpd ""`),
			errors.New(`invalid versions[""].images.masterEtcd ""`),
			errors.New(`invalid versions[""].images.genevaLogging ""`),
			errors.New(`invalid versions[""].images.genevaStatsd ""`),
			errors.New(`invalid versions[""].images.genevaTDAgent ""`),
			errors.New(`invalid versions[""].images.azureControllers ""`),
			errors.New(`invalid versions[""].images.canary ""`),
			errors.New(`invalid versions[""].images.etcdBackup ""`),
			errors.New(`invalid versions[""].images.metricsBridge ""`),
			errors.New(`invalid versions[""].images.startup ""`),
			errors.New(`invalid versions[""].images.sync ""`),
			errors.New(`invalid versions[""].images.tlsProxy ""`),
			errors.New(`invalid certificates.genevaLogging.key`),
			errors.New(`invalid certificates.genevaLogging.cert`),
			errors.New(`invalid certificates.genevaMetrics.key`),
			errors.New(`invalid certificates.genevaMetrics.cert`),
			errors.New(`invalid genevaLoggingSector ""`),
			errors.New(`invalid genevaLoggingAccount ""`),
			errors.New(`invalid genevaLoggingNamespace ""`),
			errors.New(`invalid genevaLoggingControlPlaneAccount ""`),
			errors.New(`invalid genevaLoggingControlPlaneEnvironment ""`),
			errors.New(`invalid genevaLoggingControlPlaneRegion ""`),
			errors.New(`invalid genevaMetricsAccount ""`),
			errors.New(`invalid genevaMetricsEndpoint ""`),
			errors.New(`invalid genevaImagePullSecret ""`),
			errors.New(`invalid imagePullSecret ""`),
		}

	template := pluginapi.Config{
		Versions: map[string]pluginapi.VersionConfig{
			"": {},
		},
	}
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
