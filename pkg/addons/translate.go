package addons

import (
	"context"
	"encoding/base64"
	"text/template"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	util "github.com/openshift/openshift-azure/pkg/util/template"
)

func KeyFunc(gk schema.GroupKind, namespace, name string) string {
	s := gk.String()
	if namespace != "" {
		s += "/" + namespace
	}
	s += "/" + name

	return s
}

type NestedFlags int

const (
	NestedFlagsBase64 NestedFlags = (1 << iota)
)

type authenticatedCall struct {
	cpc *cloudprovider.Config
}

func (a *authenticatedCall) getSecretFromVault(blockType, kvURL string) (string, error) {
	vaultURL, certName, err := azureclient.GetURLCertNameFromFullURL(kvURL)
	if err != nil {
		return "", err
	}
	cfg := auth.NewClientCredentialsConfig(a.cpc.AadClientID, a.cpc.AadClientSecret, a.cpc.TenantID)
	kvc, err := azureclient.NewKeyVaultClient(cfg, vaultURL)
	if err != nil {
		return "", err
	}
	bundle, err := kvc.GetSecret(context.Background(), vaultURL, certName, "")
	if err != nil {
		return "", err
	}
	return tls.GetPemBlock([]byte(*bundle.Value), blockType)
}

func translateAsset(o unstructured.Unstructured, cs *api.OpenShiftManagedCluster, ext *extra, cpc *cloudprovider.Config) (unstructured.Unstructured, error) {
	ac := authenticatedCall{cpc: cpc}

	ts := Translations[KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())]
	for _, tr := range ts {
		var s interface{}
		if tr.F != nil {
			var err error
			s, err = tr.F(cs)
			if err != nil {
				return unstructured.Unstructured{}, err
			}
		} else {
			b, err := util.Template(tr.Template,
				template.FuncMap{
					"SecretFromVault": ac.getSecretFromVault,
				}, cs, ext)
			s = string(b)
			if err != nil {
				return unstructured.Unstructured{}, err
			}
		}

		err := Translate(o.Object, tr.Path, tr.NestedPath, tr.NestedFlags, s)
		if err != nil {
			return unstructured.Unstructured{}, err
		}
	}
	return o, nil
}

func Translate(o interface{}, path jsonpath.Path, nestedPath jsonpath.Path, nestedFlags NestedFlags, v interface{}) error {
	var err error

	if nestedPath == nil {
		path.Set(o, v)
		return nil
	}

	nestedBytes := []byte(path.MustGetString(o))

	if nestedFlags&NestedFlagsBase64 != 0 {
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

	if nestedFlags&NestedFlagsBase64 != 0 {
		nestedBytes = []byte(base64.StdEncoding.EncodeToString(nestedBytes))
		if err != nil {
			panic(err)
		}
	}

	path.Set(o, string(nestedBytes))

	return nil
}

var Translations = map[string][]struct {
	Path        jsonpath.Path
	NestedPath  jsonpath.Path
	NestedFlags NestedFlags
	Template    string
	F           func(*api.OpenShiftManagedCluster) (interface{}, error)
}{
	// IMPORTANT: Translations must NOT use the quote function (i.e., write
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
			Template:   "https://{{ .Derived.PublicHostname .ContainerService }}/console/",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'console-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.masterPublicURL"),
			Template:   "https://{{ .Derived.PublicHostname .ContainerService }}",
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
	"ConfigMap/openshift-azure-logging/mdsd-config": {
		{
			Path:     jsonpath.MustCompile("$.data.'mdsd.xml'"),
			Template: "{{ .Derived.MDSDConfig .ContainerService }}",
		},
	},
	"ConfigMap/openshift-monitoring/cluster-monitoring-config": {
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusOperator.baseImage"),
			Template:   "{{ .Config.Images.PrometheusOperatorBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusOperator.prometheusConfigReloaderBaseImage"),
			Template:   "{{ .Config.Images.PrometheusConfigReloaderBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusOperator.configReloaderBaseImage"),
			Template:   "{{ .Config.Images.ConfigReloaderBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusK8s.baseImage"),
			Template:   "{{ .Config.Images.PrometheusBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.prometheusK8s.externalLabels.cluster"),
			Template:   "https://{{ .Derived.PublicHostname .ContainerService }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.alertmanagerMain.baseImage"),
			Template:   "{{ .Config.Images.AlertManagerBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.nodeExporter.baseImage"),
			Template:   "{{ .Config.Images.NodeExporterBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.grafana.baseImage"),
			Template:   "{{ .Config.Images.GrafanaBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeStateMetrics.baseImage"),
			Template:   "{{ .Config.Images.KubeStateMetricsBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeRbacProxy.baseImage"),
			Template:   "{{ .Config.Images.KubeRbacProxyBase }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.auth.baseImage"),
			Template:   "{{ .Config.Images.OAuthProxyBase }}",
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
			Template:   "{{ .ContainerService.Properties.AzProfile.ResourceGroup }}",
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
	"ConfigMap/openshift-node/node-config-compute": {
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.imageConfig.format"),
			Template:   "{{ .Config.Images.Format }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'kube-reserved'[0]"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.KubeReserved(cs, api.AgentPoolProfileRoleCompute)
			},
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'system-reserved'[0]"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.SystemReserved(cs, api.AgentPoolProfileRoleCompute)
			},
		},
	},
	"ConfigMap/openshift-node/node-config-infra": {
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.imageConfig.format"),
			Template:   "{{ .Config.Images.Format }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'kube-reserved'[0]"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.KubeReserved(cs, api.AgentPoolProfileRoleInfra)
			},
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'system-reserved'[0]"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.SystemReserved(cs, api.AgentPoolProfileRoleInfra)
			},
		},
	},
	"ConfigMap/openshift-node/node-config-master": {
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.imageConfig.format"),
			Template:   "{{ .Config.Images.Format }}",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'system-reserved'[0]"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.SystemReserved(cs, api.AgentPoolProfileRoleMaster)
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
			Template:   "https://{{ .Derived.PublicHostname .ContainerService }}/console/",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.masterPublicURL"),
			Template:   "https://{{ .Derived.PublicHostname .ContainerService }}",
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
	"DaemonSet.apps/openshift-azure-logging/mdsd": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.GenevaTDAgent }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].image"),
			Template: "{{ .Config.Images.GenevaLogging }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='SUBSCRIPTION_ID')].value"),
			Template: "{{ .ContainerService.Properties.AzProfile.SubscriptionID }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='RESOURCE_GROUP_NAME')].value"),
			Template: "{{ .ContainerService.Properties.AzProfile.ResourceGroup }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='RESOURCE_NAME')].value"),
			Template: "{{ .ContainerService.Name }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='ACCOUNT')].value"),
			Template: "{{ .Config.GenevaLoggingAccount }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='NAMESPACE')].value"),
			Template: "{{ .Config.GenevaLoggingNamespace }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='MONITORING_GCS_ACCOUNT')].value"),
			Template: "{{ .Config.GenevaLoggingControlPlaneAccount }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='MONITORING_GCS_ENVIRONMENT')].value"),
			Template: "{{ .Config.GenevaLoggingControlPlaneEnvironment }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[1].env[?(@.name='MONITORING_GCS_REGION')].value"),
			Template: "{{ .Config.GenevaLoggingControlPlaneRegion }}",
		},
	},
	"DaemonSet.apps/openshift-node/sync": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Node }}",
		},
	},
	"DaemonSet.apps/openshift-sdn/ovs": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Node }}",
		},
	},
	"DaemonSet.apps/openshift-sdn/sdn": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Node }}",
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
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='REGISTRY_HTTP_SECRET')].value"),
			Template: "{{ Base64Encode .Config.RegistryHTTPSecret }}",
		},
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
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='STATS_PASSWORD')].value"),
			Template: "{{ .Config.RouterStatsPassword }}",
		},
	},
	"Deployment.apps/default/registry-console": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='OPENSHIFT_OAUTH_PROVIDER_URL')].value"),
			Template: "https://{{ .Derived.PublicHostname .ContainerService }}",
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
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.StatsdArgs(cs)
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
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.ClusterMonitoringOperatorArgs(cs)
			},
		},
	},
	"Deployment.apps/openshift-web-console/webconsole": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.WebConsole }}",
		},
	},
	"ImageStream.image.openshift.io/openshift-node/node": {
		{
			Path:     jsonpath.MustCompile("$.spec.tags[0].from.name"),
			Template: "{{ .Config.Images.Node }}",
		},
	},
	"ImageStream.image.openshift.io/openshift-sdn/node": {
		{
			Path:     jsonpath.MustCompile("$.spec.tags[0].from.name"),
			Template: "{{ .Config.Images.Node }}",
		},
	},
	"OAuthClient.oauth.openshift.io/cockpit-oauth-client": {
		{
			Path:     jsonpath.MustCompile("$.redirectURIs[0]"),
			Template: "https://registry-console-default.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
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
			Template: "docker-registry-default.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/default/registry-console": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "registry-console-default.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/kube-service-catalog/apiserver": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "apiserver-kube-service-catalog.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-ansible-service-broker/asb-1338": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "asb-1338-openshift-ansible-service-broker.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-console/console": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "console.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
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
			Template:   "{{ .Extra.RegistryStorageAccountKey }}",
		},
	},
	"Secret/default/etc-origin-cloudprovider": {
		{
			Path: jsonpath.MustCompile("$.stringData.'azure.conf'"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				b, err := config.Derived.WorkerCloudProviderConf(cs)
				return string(b), err
			},
		},
	},
	"Secret/default/router-certs": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.crt'"),
			Template: "{{ SecretFromVault \"CERTIFICATE\" (index .ContainerService.Properties.RouterProfiles 0).RouterCertProfile.KeyVaultSecretURL }}\n{{ String (CertAsBytes .Config.Certificates.Ca.Cert) }}\n{{ SecretFromVault \"PRIVATE KEY\" (index .ContainerService.Properties.RouterProfiles 0).RouterCertProfile.KeyVaultSecretURL }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.key'"),
			Template: "{{ SecretFromVault \"PRIVATE KEY\" (index .ContainerService.Properties.RouterProfiles 0).RouterCertProfile.KeyVaultSecretURL }}",
		},
	},
	"Secret/openshift-infra/aad-group-sync-config": {
		{
			Path: jsonpath.MustCompile("$.stringData.'aad-group-sync.yaml'"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				b, err := config.Derived.AadGroupSyncConf(cs)
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
	"Secret/openshift-azure-logging/gcs-cert": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'gcscert.pem'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.GenevaLogging.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'gcskey.pem'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.GenevaLogging.Key) }}",
		},
	},
	"Secret/openshift-azure-logging/azure-registry": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'.dockerconfigjson'"),
			Template: "{{ String .Config.Images.GenevaImagePullSecret }}",
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
	"Secret/openshift-console/console-oauth-config": {
		{
			Path:     jsonpath.MustCompile("$.stringData.clientSecret"),
			Template: "{{ .Config.ConsoleOAuthSecret }}",
		},
	},
	"Service/default/router": {
		{
			Path: jsonpath.MustCompile("$.metadata.annotations['service.beta.kubernetes.io/azure-dns-label-name']"),
			F: func(cs *api.OpenShiftManagedCluster) (interface{}, error) {
				return config.Derived.RouterLBCNamePrefix(cs), nil
			},
		},
	},
	"Service/default/router-stats": {
		{
			Path:     jsonpath.MustCompile("$.metadata.annotations['prometheus.openshift.io/password']"),
			Template: "{{ .Config.RouterStatsPassword }}",
		},
	},
	"StatefulSet.apps/openshift-infra/bootstrap-autoapprover": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Node }}",
		},
	},
	"StorageClass.storage.k8s.io/azure": {
		{
			Path:     jsonpath.MustCompile("$.parameters.location"),
			Template: "{{ .ContainerService.Location }}",
		},
	},
}
