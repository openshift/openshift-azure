package api

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

var testPluginTemplateYAML = []byte(`---
genevaLoggingAccount: ccpopenshiftdiag
genevaLoggingControlPlaneAccount: RPOpenShiftAccount
genevaLoggingNamespace: CCPOpenShift
genevaLoggingSector: US-Test
genevaMetricsAccount: RPOpenShift
genevaMetricsEndpoint: https://az-int.metrics.nsatc.net/
imageOffer: osa
imagePublisher: redhat
imageSku: osa_311
imageVersion: 311.43.20181121
images:
  alertManagerBase: registry.access.redhat.com/openshift3/prometheus-alertmanager
  ansibleServiceBroker: registry.access.redhat.com/openshift3/ose-ansible-service-broker:v3.11.43
  azureControllers: quay.io/openshift-on-azure/azure-controllers:v3.11
  clusterMonitoringOperator: registry.access.redhat.com/openshift3/ose-cluster-monitoring-operator:v3.11.43
  configReloaderBase: registry.access.redhat.com/openshift3/ose-configmap-reloader
  console: registry.access.redhat.com/openshift3/ose-console:v3.11.43
  controlPlane: registry.access.redhat.com/openshift3/ose-control-plane:v3.11.43
  etcdBackup: quay.io/openshift-on-azure/etcdbackup:latest
  format: registry.access.redhat.com/openshift3/ose-${component}:${version}
  genevaImagePullSecret: e2F1dGhzOntvc2FycGludC5henVyZWNyLmlvOnt1c2VybmFtZTp1c2VybmFtZSxwYXNzd29yZDpub3R0aGVwYXNzd29yZHlvdWFyZWxvb2tpbmdmb3IsZW1haWw6b3BlbnNoaWZ0QG1pY3Jvc29mdC5jb20sYXV0aDpkWE5sY201aGJXVTZibTkwZEdobGNHRnpjM2R2Y21SNWIzVmhjbVZzYjI5cmFXNW5abTl5Q2c9PX19fQo=
  genevaLogging: osarpint.azurecr.io/acs/mdsd:12051806
  genevaStatsd: osarpint.azurecr.io/acs/mdm:git-a909a2e76
  genevaTDAgent: osarpint.azurecr.io/acs/td-agent:latest
  grafanaBase: registry.access.redhat.com/openshift3/grafana
  kubeRbacProxyBase: registry.access.redhat.com/openshift3/ose-kube-rbac-proxy
  kubeStateMetricsBase: registry.access.redhat.com/openshift3/ose-kube-state-metrics
  masterEtcd: registry.access.redhat.com/rhel7/etcd:3.2.22
  metricsBridge: quay.io/openshift-on-azure/metricsbridge:latest
  node: registry.access.redhat.com/openshift3/ose-node:v3.11.43
  nodeExporterBase: registry.access.redhat.com/openshift3/prometheus-node-exporter
  oAuthProxyBase: registry.access.redhat.com/openshift3/oauth-proxy
  prometheusBase: registry.access.redhat.com/openshift3/prometheus
  prometheusConfigReloaderBase: registry.access.redhat.com/openshift3/ose-prometheus-config-reloader
  prometheusOperatorBase: registry.access.redhat.com/openshift3/ose-prometheus-operator
  registry: registry.access.redhat.com/openshift3/ose-docker-registry:v3.11.43
  registryConsole: registry.access.redhat.com/openshift3/registry-console:v3.11.43
  router: registry.access.redhat.com/openshift3/ose-haproxy-router:v3.11.43
  serviceCatalog: registry.access.redhat.com/openshift3/ose-service-catalog:v3.11.43
  sync: quay.io/openshift-on-azure/sync:latest
  templateServiceBroker: registry.access.redhat.com/openshift3/ose-template-service-broker:v3.11.43
  webConsole: registry.access.redhat.com/openshift3/ose-web-console:v3.11.43
runningUnderTest: true
certificates:
   genevaLogging:
     key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlCT1FJQkFBSkJBTko2cWhjWmlBK0tsUURETlZqQTY0TVRJbSt3WGFWUnZ6Q2Zwbm9ya0Y0OVJHMVYvVm5mClZCTmVPNTBvb3E1ZGNpcElOM284bmVwY09QQU5Ybk5vVkVNQ0F3RUFBUUpBSFNIclR2MHlydXdBaWJWN09jaWkKRUdkaW1kRHdkVVJtVVNXWDFrc1hWV09uTXFxeFk4c1ZEZTQrOVNqbW1uMHRpZjc3UDRHWE0zUWxKSjFXa0tvQQo4UUloQVBPWjhjRDd0NTNBazIzOWh1bytMR1FnNUZZaVdVM0JGWTJ1VUQ0RG1EL0xBaUVBM1RFbHdFcC8ybXN5CkVlaXNlc3B6ZlBqQXVSME16clRoS3FEcTEwa3BQbWtDSUdFaThORElUd2FicE81R0cwZEt0WDdUMHRrNTV5eG4KSXdZVkRUQTlWTGVUQWlBd2dhcXB0S3k5Rld6eGlIanFwS01XOE9ZeXNqQXcxSEhjaTFWMHlOS0dvUUlnWlZiVQpNZU1kQVdVdkVJbXowY0RnQ3BLTCtqNDAySm1iZFZ1dkhNNyt3QVU9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
     cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUI2akNDQVpTZ0F3SUJBZ0lKQVBaMC8ydDJqYXJ5TUEwR0NTcUdTSWIzRFFFQkN3VUFNQlV4RXpBUkJnTlYKQkFNVENtdDFZbVZ5Ym1WMFpYTXdIaGNOTVRneE1URXlNVGN5TWpJeFdoY05NakF4TVRFeU1UY3lNakl4V2pBVgpNUk13RVFZRFZRUURFd3ByZFdKbGNtNWxkR1Z6TUZ3d0RRWUpLb1pJaHZjTkFRRUJCUUFEU3dBd1NBSkJBTko2CnFoY1ppQStLbFFERE5WakE2NE1USW0rd1hhVlJ2ekNmcG5vcmtGNDlSRzFWL1ZuZlZCTmVPNTBvb3E1ZGNpcEkKTjNvOG5lcGNPUEFOWG5Ob1ZFTUNBd0VBQWFPQnhqQ0J3ekFPQmdOVkhROEJBZjhFQkFNQ0JhQXdEQVlEVlIwVApBUUgvQkFJd0FEQ0JvZ1lEVlIwUkJJR2FNSUdYZ2dwcmRXSmxjbTVsZEdWemdnMXRZWE4wWlhJdE1EQXdNREF3CmdnMXRZWE4wWlhJdE1EQXdNREF4Z2cxdFlYTjBaWEl0TURBd01EQXlnZ3ByZFdKbGNtNWxkR1Z6Z2hKcmRXSmwKY201bGRHVnpMbVJsWm1GMWJIU0NGbXQxWW1WeWJtVjBaWE11WkdWbVlYVnNkQzV6ZG1PQ0pHdDFZbVZ5Ym1WMApaWE11WkdWbVlYVnNkQzV6ZG1NdVkyeDFjM1JsY2k1c2IyTmhiREFOQmdrcWhraUc5dzBCQVFzRkFBTkJBTkF2CjUwaXJlRHRqUkRiMVZydjJEcmRGZXhrT2hhZzNJM3dEVWlPb0loYjVuTkNNRjdnMEF1VEpOWkhFUXFOWDNYb1gKYkZpRkExdUxLZEZMc1B1T1dTUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
   genevaMetrics:
     key: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlCT1FJQkFBSkJBTko2cWhjWmlBK0tsUURETlZqQTY0TVRJbSt3WGFWUnZ6Q2Zwbm9ya0Y0OVJHMVYvVm5mClZCTmVPNTBvb3E1ZGNpcElOM284bmVwY09QQU5Ybk5vVkVNQ0F3RUFBUUpBSFNIclR2MHlydXdBaWJWN09jaWkKRUdkaW1kRHdkVVJtVVNXWDFrc1hWV09uTXFxeFk4c1ZEZTQrOVNqbW1uMHRpZjc3UDRHWE0zUWxKSjFXa0tvQQo4UUloQVBPWjhjRDd0NTNBazIzOWh1bytMR1FnNUZZaVdVM0JGWTJ1VUQ0RG1EL0xBaUVBM1RFbHdFcC8ybXN5CkVlaXNlc3B6ZlBqQXVSME16clRoS3FEcTEwa3BQbWtDSUdFaThORElUd2FicE81R0cwZEt0WDdUMHRrNTV5eG4KSXdZVkRUQTlWTGVUQWlBd2dhcXB0S3k5Rld6eGlIanFwS01XOE9ZeXNqQXcxSEhjaTFWMHlOS0dvUUlnWlZiVQpNZU1kQVdVdkVJbXowY0RnQ3BLTCtqNDAySm1iZFZ1dkhNNyt3QVU9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
     cert: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUI2akNDQVpTZ0F3SUJBZ0lKQVBaMC8ydDJqYXJ5TUEwR0NTcUdTSWIzRFFFQkN3VUFNQlV4RXpBUkJnTlYKQkFNVENtdDFZbVZ5Ym1WMFpYTXdIaGNOTVRneE1URXlNVGN5TWpJeFdoY05NakF4TVRFeU1UY3lNakl4V2pBVgpNUk13RVFZRFZRUURFd3ByZFdKbGNtNWxkR1Z6TUZ3d0RRWUpLb1pJaHZjTkFRRUJCUUFEU3dBd1NBSkJBTko2CnFoY1ppQStLbFFERE5WakE2NE1USW0rd1hhVlJ2ekNmcG5vcmtGNDlSRzFWL1ZuZlZCTmVPNTBvb3E1ZGNpcEkKTjNvOG5lcGNPUEFOWG5Ob1ZFTUNBd0VBQWFPQnhqQ0J3ekFPQmdOVkhROEJBZjhFQkFNQ0JhQXdEQVlEVlIwVApBUUgvQkFJd0FEQ0JvZ1lEVlIwUkJJR2FNSUdYZ2dwcmRXSmxjbTVsZEdWemdnMXRZWE4wWlhJdE1EQXdNREF3CmdnMXRZWE4wWlhJdE1EQXdNREF4Z2cxdFlYTjBaWEl0TURBd01EQXlnZ3ByZFdKbGNtNWxkR1Z6Z2hKcmRXSmwKY201bGRHVnpMbVJsWm1GMWJIU0NGbXQxWW1WeWJtVjBaWE11WkdWbVlYVnNkQzV6ZG1PQ0pHdDFZbVZ5Ym1WMApaWE11WkdWbVlYVnNkQzV6ZG1NdVkyeDFjM1JsY2k1c2IyTmhiREFOQmdrcWhraUc5dzBCQVFzRkFBTkJBTkF2CjUwaXJlRHRqUkRiMVZydjJEcmRGZXhrT2hhZzNJM3dEVWlPb0loYjVuTkNNRjdnMEF1VEpOWkhFUXFOWDNYb1gKYkZpRkExdUxLZEZMc1B1T1dTUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
`)

func TestPluginTemplateValidate(t *testing.T) {
	tests := map[string]struct {
		f            func(*pluginapi.Config)
		expectedErrs []error
	}{
		"test yaml parsing": { // test yaml parsing

		},
		"invalid ImageOffer": {
			f: func(t *pluginapi.Config) {
				t.ImageOffer = "non-osa"
			},
			expectedErrs: []error{errors.New(`imageOffer should be osa`)},
		},
		"invalid ImagePublisher": {
			f: func(t *pluginapi.Config) {
				t.ImagePublisher = "microsoft"
			},
			expectedErrs: []error{errors.New(`imagePublisher should be redhat`)},
		},
		"invalid ImageSKU": {
			f: func(t *pluginapi.Config) {
				t.ImageSKU = "osa_310"
			},
			expectedErrs: []error{errors.New(`invalid ImageSKU "osa_310"`)},
		},
		"invalid ImageVersion": {
			f: func(t *pluginapi.Config) {
				t.ImageVersion = "1.23.456"
			},
			expectedErrs: []error{errors.New(`invalid ImageVersion "1.23.456"`)},
		},
		"empty GenevaLoggingSector": {
			f: func(t *pluginapi.Config) {
				t.GenevaLoggingSector = ""
			},
			expectedErrs: []error{errors.New(`genevaLoggingSector cannot be empty`)},
		},
		"empty GenevaLoggingAccount": {
			f: func(t *pluginapi.Config) {
				t.GenevaLoggingAccount = ""
			},
			expectedErrs: []error{errors.New(`genevaLoggingAccount cannot be empty`)},
		},
		"empty GenevaLoggingNamespace": {
			f: func(t *pluginapi.Config) {
				t.GenevaLoggingNamespace = ""
			},
			expectedErrs: []error{errors.New(`genevaLoggingNamespace cannot be empty`)},
		},
		"empty GenevaLoggingControlPlaneAccount": {
			f: func(t *pluginapi.Config) {
				t.GenevaLoggingControlPlaneAccount = ""
			},
			expectedErrs: []error{errors.New(`genevaLoggingControlPlaneAccount cannot be empty`)},
		},
		"empty GenevaMetricsAccount": {
			f: func(t *pluginapi.Config) {
				t.GenevaMetricsAccount = ""
			},
			expectedErrs: []error{errors.New(`genevaMetricsAccount cannot be empty`)},
		},
		"empty GenevaMetricsEndpoint": {
			f: func(t *pluginapi.Config) {
				t.GenevaMetricsEndpoint = ""
			},
			expectedErrs: []error{errors.New(`genevaMetricsEndpoint cannot be empty`)},
		},
		"invalid GenevaLogging key": {
			f: func(t *pluginapi.Config) {
				t.Certificates.GenevaLogging.Key = nil
			},
			expectedErrs: []error{errors.New(`invalid GenevaLogging key`)},
		},
		"invalid GenevaLogging cert": {
			f: func(t *pluginapi.Config) {
				t.Certificates.GenevaLogging.Cert = nil
			},
			expectedErrs: []error{errors.New(`invalid GenevaLogging certificate`)},
		},
		"invalid GenevaMetrics key": {
			f: func(t *pluginapi.Config) {
				t.Certificates.GenevaMetrics.Key = nil
			},
			expectedErrs: []error{errors.New(`invalid GenevaMetrics key`)},
		},
		"invalid GenevaMetrics cert": {
			f: func(t *pluginapi.Config) {
				t.Certificates.GenevaMetrics.Cert = nil
			},
			expectedErrs: []error{errors.New(`invalid GenevaMetrics certificate`)},
		},
		"empty image.Format": {
			f: func(t *pluginapi.Config) {
				t.Images.Format = ""
			},
			expectedErrs: []error{errors.New(`images.Format cannot be empty`)},
		},
		"empty image.GenevaImagePullSecret": {
			f: func(t *pluginapi.Config) {
				t.Images.GenevaImagePullSecret = []byte{}
			},
			expectedErrs: []error{errors.New(`images.GenevaImagePullSecret cannot be empty`)},
		},
		"empty image.ClusterMonitoringOperator": {
			f: func(t *pluginapi.Config) {
				t.Images.ClusterMonitoringOperator = ""
			},
			expectedErrs: []error{errors.New(`images.ClusterMonitoringOperator cannot be empty`)},
		},
		"empty image.AzureControllers": {
			f: func(t *pluginapi.Config) {
				t.Images.AzureControllers = ""
			},
			expectedErrs: []error{errors.New(`images.AzureControllers cannot be empty`)},
		},
		"empty image.PrometheusOperatorBase": {
			f: func(t *pluginapi.Config) {
				t.Images.PrometheusOperatorBase = ""
			},
			expectedErrs: []error{errors.New(`images.PrometheusOperatorBase cannot be empty`)},
		},
		"empty image.PrometheusBase": {
			f: func(t *pluginapi.Config) {
				t.Images.PrometheusBase = ""
			},
			expectedErrs: []error{errors.New(`images.PrometheusBase cannot be empty`)},
		},
		"empty image.PrometheusConfigReloaderBase": {
			f: func(t *pluginapi.Config) {
				t.Images.PrometheusConfigReloaderBase = ""
			},
			expectedErrs: []error{errors.New(`images.PrometheusConfigReloaderBase cannot be empty`)},
		},
		"empty image.ConfigReloaderBase": {
			f: func(t *pluginapi.Config) {
				t.Images.ConfigReloaderBase = ""
			},
			expectedErrs: []error{errors.New(`images.ConfigReloaderBase cannot be empty`)},
		},
		"empty image.AlertManagerBase": {
			f: func(t *pluginapi.Config) {
				t.Images.AlertManagerBase = ""
			},
			expectedErrs: []error{errors.New(`images.AlertManagerBase cannot be empty`)},
		},
		"empty image.NodeExporterBase": {
			f: func(t *pluginapi.Config) {
				t.Images.NodeExporterBase = ""
			},
			expectedErrs: []error{errors.New(`images.NodeExporterBase cannot be empty`)},
		},
		"empty image.GrafanaBase": {
			f: func(t *pluginapi.Config) {
				t.Images.GrafanaBase = ""
			},
			expectedErrs: []error{errors.New(`images.GrafanaBase cannot be empty`)},
		},
		"empty image.KubeStateMetricsBase": {
			f: func(t *pluginapi.Config) {
				t.Images.KubeStateMetricsBase = ""
			},
			expectedErrs: []error{errors.New(`images.KubeStateMetricsBase cannot be empty`)},
		},
		"empty image.KubeRbacProxyBase": {
			f: func(t *pluginapi.Config) {
				t.Images.KubeRbacProxyBase = ""
			},
			expectedErrs: []error{errors.New(`images.KubeRbacProxyBase cannot be empty`)},
		},
		"empty image.OAuthProxyBase": {
			f: func(t *pluginapi.Config) {
				t.Images.OAuthProxyBase = ""
			},
			expectedErrs: []error{errors.New(`images.OAuthProxyBase cannot be empty`)},
		},
		"empty image.MasterEtcd": {
			f: func(t *pluginapi.Config) {
				t.Images.MasterEtcd = ""
			},
			expectedErrs: []error{errors.New(`images.MasterEtcd cannot be empty`)},
		},
		"empty image.ControlPlane": {
			f: func(t *pluginapi.Config) {
				t.Images.ControlPlane = ""
			},
			expectedErrs: []error{errors.New(`images.ControlPlane cannot be empty`)},
		},
		"empty image.Node": {
			f: func(t *pluginapi.Config) {
				t.Images.Node = ""
			},
			expectedErrs: []error{errors.New(`images.Node cannot be empty`)},
		},
		"empty image.ServiceCatalog": {
			f: func(t *pluginapi.Config) {
				t.Images.ServiceCatalog = ""
			},
			expectedErrs: []error{errors.New(`images.ServiceCatalog cannot be empty`)},
		},
		"empty image.Sync": {
			f: func(t *pluginapi.Config) {
				t.Images.Sync = ""
			},
			expectedErrs: []error{errors.New(`images.Sync cannot be empty`)},
		},
		"empty image.TemplateServiceBroker": {
			f: func(t *pluginapi.Config) {
				t.Images.TemplateServiceBroker = ""
			},
			expectedErrs: []error{errors.New(`images.TemplateServiceBroker cannot be empty`)},
		},
		"empty image.Registry": {
			f: func(t *pluginapi.Config) {
				t.Images.Registry = ""
			},
			expectedErrs: []error{errors.New(`images.Registry cannot be empty`)},
		},
		"empty image.Router": {
			f: func(t *pluginapi.Config) {
				t.Images.Router = ""
			},
			expectedErrs: []error{errors.New(`images.Router cannot be empty`)},
		},
		"empty image.RegistryConsole": {
			f: func(t *pluginapi.Config) {
				t.Images.RegistryConsole = ""
			},
			expectedErrs: []error{errors.New(`images.RegistryConsole cannot be empty`)},
		},
		"empty image.AnsibleServiceBroker": {
			f: func(t *pluginapi.Config) {
				t.Images.AnsibleServiceBroker = ""
			},
			expectedErrs: []error{errors.New(`images.AnsibleServiceBroker cannot be empty`)},
		},
		"empty image.WebConsole": {
			f: func(t *pluginapi.Config) {
				t.Images.WebConsole = ""
			},
			expectedErrs: []error{errors.New(`images.WebConsole cannot be empty`)},
		},
		"empty image.Console": {
			f: func(t *pluginapi.Config) {
				t.Images.Console = ""
			},
			expectedErrs: []error{errors.New(`images.Console cannot be empty`)},
		},
		"empty image.EtcdBackup": {
			f: func(t *pluginapi.Config) {
				t.Images.EtcdBackup = ""
			},
			expectedErrs: []error{errors.New(`images.EtcdBackup cannot be empty`)},
		},
	}

	for name, test := range tests {
		var template *pluginapi.Config
		err := yaml.Unmarshal(testPluginTemplateYAML, &template)
		if err != nil {
			t.Fatal(err)
		}
		if test.f != nil {
			test.f(template)
		}
		v := Validator{}
		errs := v.ValidatePluginTemplate(template)
		if !reflect.DeepEqual(errs, test.expectedErrs) {
			t.Logf("test case %q", name)
			t.Errorf("expected errors:")
			for _, err := range test.expectedErrs {
				t.Errorf("\t%v", err)
			}
			t.Error("received errors:")
			for _, err := range errs {
				t.Errorf("\t%v", err)
			}
		}
	}
}
