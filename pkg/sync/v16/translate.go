package sync

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	util "github.com/openshift/openshift-azure/pkg/util/template"
)

func keyFunc(gk schema.GroupKind, namespace, name string) string {
	s := gk.String()
	if namespace != "" {
		s += "/" + namespace
	}
	s += "/" + name

	return s
}

type nestedFlags int

const (
	nestedFlagsBase64 nestedFlags = (1 << iota)
)

func translateAsset(o unstructured.Unstructured, cs *api.OpenShiftManagedCluster) (unstructured.Unstructured, error) {
	ts := translations[keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())]
	for i, tr := range ts {
		var s interface{}
		if tr.F != nil {
			var err error
			s, err = tr.F(cs, o.Object)
			if err != nil {
				return unstructured.Unstructured{}, err
			}
		} else {
			b, err := util.Template(fmt.Sprintf("%s/%d", keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName()), i), tr.Template, nil, map[string]interface{}{
				"ContainerService": cs,
				"Config":           &cs.Config,
				"Derived":          derived,
			})
			s = string(b)
			if err != nil {
				return unstructured.Unstructured{}, err
			}
		}

		err := translate(o.Object, tr.Path, tr.NestedPath, tr.nestedFlags, s)
		if err != nil {
			return unstructured.Unstructured{}, err
		}
	}
	return o, nil
}

func translate(o interface{}, path jsonpath.Path, nestedPath jsonpath.Path, nestedFlags nestedFlags, v interface{}) error {
	var err error

	if nestedPath == nil {
		path.Set(o, v)
		return nil
	}

	nestedBytes := []byte(path.MustGetString(o))

	if nestedFlags&nestedFlagsBase64 != 0 {
		nestedBytes, err = base64.StdEncoding.DecodeString(string(nestedBytes))
		if err != nil {
			return err
		}
	}

	var nestedObject interface{}
	err = yaml.Unmarshal(nestedBytes, &nestedObject)
	if err != nil {
		panic(err)
	}

	nestedPath.Set(nestedObject, v)

	nestedBytes, err = yaml.Marshal(nestedObject)
	if err != nil {
		panic(err)
	}

	if nestedFlags&nestedFlagsBase64 != 0 {
		nestedBytes = []byte(base64.StdEncoding.EncodeToString(nestedBytes))
		if err != nil {
			panic(err)
		}
	}

	path.Set(o, string(nestedBytes))

	return nil
}

var translations = map[string][]struct {
	Path        jsonpath.Path
	NestedPath  jsonpath.Path
	nestedFlags nestedFlags
	Template    string
	F           func(*api.OpenShiftManagedCluster, interface{}) (interface{}, error)
}{
	// IMPORTANT: translations must NOT use the quote function (i.e., write
	// "{{ .Config.Foo }}", NOT "{{ .Config.Foo | quote }}").  This is because
	// the translations operate on in-memory objects, not on serialised YAML.
	// Correct quoting will be handled automatically by the marshaller.
	"APIService.apiregistration.k8s.io/v1beta1.servicecatalog.k8s.io": {
		{
			Path:     jsonpath.MustCompile("$.spec.caBundle"),
			Template: "{{ Base64Encode (CertAsBytes .Config.Certificates.ServiceCatalogCa.Cert) }}",
		},
	},
	"ClusterServiceBroker.servicecatalog.k8s.io/ansible-service-broker": {
		{
			Path:     jsonpath.MustCompile("$.spec.caBundle"),
			Template: "{{ Base64Encode (CertAsBytes .Config.Certificates.ServiceSigningCa.Cert) }}",
		},
	},
	"ClusterServiceBroker.servicecatalog.k8s.io/template-service-broker": {
		{
			Path:     jsonpath.MustCompile("$.spec.caBundle"),
			Template: "{{ Base64Encode (CertAsBytes .Config.Certificates.ServiceSigningCa.Cert) }}",
		},
	},
	"ConfigMap/kube-service-catalog/cluster-info": {
		{
			Path:     jsonpath.MustCompile("$.data.id"),
			Template: "{{ .Config.ServiceCatalogClusterID }}",
		},
	},
	"ConfigMap/kube-system/extension-apiserver-authentication": {
		{
			Path:     jsonpath.MustCompile("$.data.'client-ca-file'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.Ca.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.data.'requestheader-client-ca-file'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.FrontProxyCa.Cert) }}",
		},
	},
	"ConfigMap/openshift-console/console-config": {
		{
			Path:       jsonpath.MustCompile("$.data.'console-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.consoleBaseAddress"),
			Template:   "https://console.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'console-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.developerConsolePublicURL"),
			Template:   "https://{{ .ContainerService.Properties.PublicHostname }}/console/",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'console-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.masterPublicURL"),
			Template:   "https://{{ .ContainerService.Properties.PublicHostname }}",
		},
	},
	"ConfigMap/openshift-ansible-service-broker/broker-config": {
		{
			Path:       jsonpath.MustCompile("$.data.'broker-config'"),
			NestedPath: jsonpath.MustCompile("$.registry[?(@.type='rhcc')].url"),
			Template:   "https://{{ .Derived.RegistryURL .ContainerService }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'broker-config'"),
			NestedPath: jsonpath.MustCompile("$.registry[?(@.type='rhcc')].tag"),
			Template:   "{{ .Derived.OpenShiftVersionTag .ContainerService }}",
		},
	},
	"ConfigMap/openshift-monitoring/cluster-monitoring-config": {
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusOperator.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.PrometheusOperator }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusOperator.prometheusConfigReloaderBaseImage"),
			Template:   "{{ ImageOnly .Config.Images.PrometheusConfigReloader }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusOperator.configReloaderBaseImage"),
			Template:   "{{ ImageOnly .Config.Images.ConfigReloader }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusK8s.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.Prometheus }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusK8s.externalLabels.cluster"),
			Template:   "https://{{ .ContainerService.Properties.PublicHostname }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.alertmanagerMain.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.AlertManager }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.nodeExporter.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.NodeExporter }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.grafana.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.Grafana }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeStateMetrics.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.KubeStateMetrics }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeRbacProxy.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.KubeRbacProxy }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.auth.baseImage"),
			Template:   "{{ ImageOnly .Config.Images.OAuthProxy }}",
		},
	},
	"ConfigMap/openshift-azure-monitoring/metrics-bridge": {
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.account"),
			Template:   "{{ .Config.GenevaMetricsAccount }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.region"),
			Template:   "{{ .ContainerService.Location }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.resourceGroupName"),
			F: func(cs *api.OpenShiftManagedCluster, o interface{}) (interface{}, error) {
				res, err := azure.ParseResourceID(cs.ID)
				return res.ResourceGroup, err
			},
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.resourceName"),
			Template:   "{{ .ContainerService.Name }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.subscriptionId"),
			Template:   "{{ .ContainerService.Properties.AzProfile.SubscriptionID }}",
		},
	},
	"ConfigMap/openshift-azure-branding/branding": {
		{
			Path: jsonpath.MustCompile("$.data.'branding.js'"),
			F: func(cs *api.OpenShiftManagedCluster, o interface{}) (interface{}, error) {
				ver, err := derived.OpenShiftClientVersion(cs)
				if err != nil {
					return nil, err
				}
				bjs := jsonpath.MustCompile("$.data.'branding.js'")
				return strings.Replace(bjs.MustGetString(o), "{VERSION}", ver, -1), nil
			},
		},
	},
	"ConfigMap/openshift-web-console/webconsole-config": {
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.adminConsolePublicURL"),
			Template:   "https://console.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.consolePublicURL"),
			Template:   "https://{{ .ContainerService.Properties.PublicHostname }}/console/",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.masterPublicURL"),
			Template:   "https://{{ .ContainerService.Properties.PublicHostname }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.extensions.stylesheetURLs[0]"),
			Template:   "https://branding.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}/branding.css",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.extensions.scriptURLs[0]"),
			Template:   "https://branding.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}/branding.js",
		},
	},
	"CronJob.batch/openshift-etcd/etcd-backup": {
		{
			Path:     jsonpath.MustCompile("$.spec.jobTemplate.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.EtcdBackup }}",
		},
	},
	"DaemonSet.apps/kube-service-catalog/apiserver": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.ServiceCatalog }}",
		},
	},
	"DaemonSet.apps/kube-service-catalog/controller-manager": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.ServiceCatalog }}",
		},
	},
	"DaemonSet.apps/openshift-azure-logging/omsagent": {
		{
			Path:     jsonpath.MustCompile("$.metadata.labels['azure.openshift.io/sync-pod-optionally-apply']"),
			Template: "{{ .ContainerService.Properties.MonitorProfile.Enabled }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.LogAnalyticsAgent }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='AKS_RESOURCE_ID')].value"),
			Template: "{{ .ContainerService.ID }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='AKS_REGION')].value"),
			Template: "{{ .ContainerService.Location }}",
		},
	},
	"DaemonSet.apps/openshift-template-service-broker/apiserver": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.TemplateServiceBroker }}",
		},
	},
	"DaemonSet.apps/default/docker-registry": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Registry }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.initContainers[0].env[?(@.name='REGISTRY_STORAGE_ACCOUNT_NAME')].value"),
			Template: "{{ .Config.RegistryStorageAccount }}",
		},
	},
	"DaemonSet.apps/default/router": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Router }}",
		},
	},
	"DaemonSet.apps/openshift-azure-monitoring/etcd-metrics": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.TLSProxy }}",
		},
	},
	"Deployment.apps/openshift-azure-logging/omsagent-rs": {
		{
			Path:     jsonpath.MustCompile("$.metadata.labels['azure.openshift.io/sync-pod-optionally-apply']"),
			Template: "{{ .ContainerService.Properties.MonitorProfile.Enabled }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.LogAnalyticsAgent }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='AKS_RESOURCE_ID')].value"),
			Template: "{{ .ContainerService.ID }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='AKS_REGION')].value"),
			Template: "{{ .ContainerService.Location }}",
		},
	},
	"Deployment.apps/default/registry-console": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='OPENSHIFT_OAUTH_PROVIDER_URL')].value"),
			Template: "https://{{ .ContainerService.Properties.PublicHostname }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='REGISTRY_HOST')].value"),
			Template: "docker-registry-default.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.RegistryConsole }}",
		},
	},
	"Deployment.apps/openshift-ansible-service-broker/asb": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.AnsibleServiceBroker }}",
		},
	},
	"Deployment.apps/openshift-azure-monitoring/metrics-bridge": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='statsd')].image"),
			Template: "{{ .Config.Images.GenevaStatsd }}",
		},
		{
			Path: jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='statsd')].args"),
			F: func(cs *api.OpenShiftManagedCluster, o interface{}) (interface{}, error) {
				return derived.StatsdArgs(cs)
			},
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='metricsbridge')].image"),
			Template: "{{ .Config.Images.MetricsBridge }}",
		},
	},
	"Deployment.apps/openshift-console/console": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Console }}",
		},
	},
	"Deployment.apps/openshift-azure-branding/branding": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Httpd }}",
		},
	},
	"Deployment.apps/openshift-infra/customer-admin-controller": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.AzureControllers }}",
		},
	},
	"Deployment.apps/openshift-monitoring/cluster-monitoring-operator": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.ClusterMonitoringOperator }}",
		},
		{
			Path: jsonpath.MustCompile("$.spec.template.spec.containers[0].args"),
			F: func(cs *api.OpenShiftManagedCluster, o interface{}) (interface{}, error) {
				return derived.ClusterMonitoringOperatorArgs(cs)
			},
		},
	},
	"Deployment.apps/openshift-monitoring/metrics-server": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.MetricsServer}}",
		},
	},
	"Deployment.apps/openshift-web-console/webconsole": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.WebConsole }}",
		},
	},
	"OAuthClient.oauth.openshift.io/cockpit-oauth-client": {
		{
			Path:     jsonpath.MustCompile("$.redirectURIs[0]"),
			Template: "https://registry-console.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
		{
			Path:     jsonpath.MustCompile("$.secret"),
			Template: "{{ .Config.RegistryConsoleOAuthSecret }}",
		},
	},
	"OAuthClient.oauth.openshift.io/openshift-console": {
		{
			Path:     jsonpath.MustCompile("$.redirectURIs[0]"),
			Template: "https://console.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
		{
			Path:     jsonpath.MustCompile("$.secret"),
			Template: "{{ .Config.ConsoleOAuthSecret }}",
		},
	},
	"Route.route.openshift.io/default/docker-registry": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "docker-registry.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/default/registry-console": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "registry-console.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-console/console": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "console.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-azure-branding/branding": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "branding.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-azure-monitoring/canary": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "canary-openshift-azure-monitoring.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Secret/default/registry-certificates": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'registry.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.Registry.Cert) }}\n{{ String (CertAsBytes .Config.Certificates.Ca.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'registry.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.Registry.Key) }}",
		},
	},
	"Secret/default/registry-config": {
		{
			Path:       jsonpath.MustCompile("$.stringData.'config.yml'"),
			NestedPath: jsonpath.MustCompile("$.storage.azure.accountname"),
			Template:   "{{ .Config.RegistryStorageAccount }}",
		},
		{
			Path:       jsonpath.MustCompile("$.stringData.'config.yml'"),
			NestedPath: jsonpath.MustCompile("$.storage.azure.accountkey"),
			Template:   "{{ .Config.RegistryStorageAccountKey }}",
		},
	},
	"Secret/default/registry-console": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.cert'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.RegistryConsole.Cert) }}\n{{ String (PrivateKeyAsBytes .Config.Certificates.RegistryConsole.Key) }}",
		},
	},
	"Secret/default/router-certs": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.crt'"),
			Template: "{{ String (CertChainAsBytes .Config.Certificates.Router.Certs) }}\n{{ String (PrivateKeyAsBytes .Config.Certificates.Router.Key) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.Router.Key) }}",
		},
	},
	"Secret/default/docker-registry-http": {
		{
			Path:     jsonpath.MustCompile("$.stringData.password"),
			Template: "{{ Base64Encode .Config.RegistryHTTPSecret }}",
		},
	},
	"Secret/default/router-stats": {
		{
			Path:     jsonpath.MustCompile("$.stringData.password"),
			Template: "{{ .Config.RouterStatsPassword }}",
		},
	},
	"Secret/openshift-infra/aad-group-sync-config": {
		{
			Path: jsonpath.MustCompile("$.stringData.'aad-group-sync.yaml'"),
			F: func(cs *api.OpenShiftManagedCluster, o interface{}) (interface{}, error) {
				b, err := derived.AadGroupSyncConf(cs)
				return string(b), err
			},
		},
	},
	"Secret/kube-service-catalog/apiserver-ssl": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.ServiceCatalogServer.Cert) }}\n{{ String (CertAsBytes .Config.Certificates.ServiceCatalogCa.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.ServiceCatalogServer.Key) }}",
		},
	},
	"Secret/openshift/redhat-registry": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'.dockerconfigjson'"),
			Template: "{{ String .Config.Images.ImagePullSecret }}",
		},
	},
	"Secret/openshift-azure-logging/omsagent-secret": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'WSID'"),
			Template: "{{ .ContainerService.Properties.MonitorProfile.WorkspaceID }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'KEY'"),
			Template: "{{ .ContainerService.Properties.MonitorProfile.WorkspaceKey }}",
		},
	},
	"Secret/openshift-azure-monitoring/azure-registry": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'.dockerconfigjson'"),
			Template: "{{ String .Config.Images.GenevaImagePullSecret }}",
		},
	},
	"Secret/openshift-azure-monitoring/mdm-cert": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'cert.pem'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.GenevaMetrics.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'key.pem'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.GenevaMetrics.Key) }}",
		},
	},
	"Secret/openshift-azure-monitoring/etcd-metrics": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'username'"),
			Template: "{{ .Config.EtcdMetricsUsername }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'password'"),
			Template: "{{ .Config.EtcdMetricsPassword }}",
		},
	},
	"Secret/openshift-console/console-oauth-config": {
		{
			Path:     jsonpath.MustCompile("$.stringData.clientSecret"),
			Template: "{{ .Config.ConsoleOAuthSecret }}",
		},
	},
	"Secret/openshift-monitoring/router-stats": {
		{
			Path:     jsonpath.MustCompile("$.stringData.password"),
			Template: "{{ .Config.RouterStatsPassword }}",
		},
	},
	"Secret/openshift-monitoring/etcd-metrics": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'username'"),
			Template: "{{ .Config.EtcdMetricsUsername }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'password'"),
			Template: "{{ .Config.EtcdMetricsPassword }}",
		},
	},
	"Secret/openshift-monitoring/metrics-server-certs": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.MetricsServer.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.MetricsServer.Key) }}",
		},
	},
	"Service/default/router": {
		{
			Path: jsonpath.MustCompile("$.metadata.annotations['service.beta.kubernetes.io/azure-dns-label-name']"),
			F: func(cs *api.OpenShiftManagedCluster, o interface{}) (interface{}, error) {
				return derived.RouterLBCNamePrefix(cs), nil
			},
		},
	},
	"StatefulSet.apps/openshift-infra/bootstrap-autoapprover": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Node }}",
		},
	},
	"StatefulSet.apps/openshift-azure-monitoring/canary": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Canary }}",
		},
	},
	"StorageClass.storage.k8s.io/azure-disk": {
		{
			Path:     jsonpath.MustCompile("$.parameters.location"),
			Template: "{{ .ContainerService.Location }}",
		},
	},
	"StorageClass.storage.k8s.io/azure-file": {
		{
			Path:     jsonpath.MustCompile("$.parameters.storageAccount"),
			Template: "{{ .Config.AzureFileStorageAccount }}",
		},
	},
}
