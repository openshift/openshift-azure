package addons

import (
	"encoding/base64"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util"
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

func translateAsset(o unstructured.Unstructured, cs *acsapi.OpenShiftManagedCluster, ext *extra) (unstructured.Unstructured, error) {
	ts := Translations[KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())]
	for _, tr := range ts {
		var s string
		if tr.F != nil {
			var err error
			s, err = tr.F(cs)
			if err != nil {
				return unstructured.Unstructured{}, err
			}
		} else {
			b, err := util.Template(tr.Template, nil, cs, ext)
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

func Translate(o interface{}, path jsonpath.Path, nestedPath jsonpath.Path, nestedFlags NestedFlags, v string) error {
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
	F           func(*acsapi.OpenShiftManagedCluster) (string, error)
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
	"ConfigMap/openshift-node/node-config-compute": {
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.imageConfig.format"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				return config.Derived.ImageConfigFormat(cs), nil
			},
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'kube-reserved'[0]"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				return config.Derived.KubeReserved(cs, acsapi.AgentPoolProfileRoleCompute), nil
			},
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'system-reserved'[0]"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				return config.Derived.SystemReserved(cs, acsapi.AgentPoolProfileRoleCompute), nil
			},
		},
	},
	"ConfigMap/openshift-node/node-config-infra": {
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.imageConfig.format"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				return config.Derived.ImageConfigFormat(cs), nil
			},
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'kube-reserved'[0]"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				return config.Derived.KubeReserved(cs, acsapi.AgentPoolProfileRoleInfra), nil
			},
		},
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.kubeletArguments.'system-reserved'[0]"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				return config.Derived.SystemReserved(cs, acsapi.AgentPoolProfileRoleInfra), nil
			},
		},
	},
	"ConfigMap/openshift-node/node-config-master": {
		{
			Path:       jsonpath.MustCompile("$.data.'node-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.imageConfig.format"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				return config.Derived.ImageConfigFormat(cs), nil
			},
		},
	},
	"ConfigMap/openshift-web-console/webconsole-config": {
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
	"CronJob/openshift-etcd/etcd-backup": {
		{
			Path:     jsonpath.MustCompile("$.spec.jobTemplate.spec.template.containers[0].image"),
			Template: "{{ .Config.Images.Cli }}",
		},
	},
	"DaemonSet.apps/openshift-metrics/prometheus-node-exporter": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.PrometheusNodeExporter }}",
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
	"Deployment.apps/default/docker-registry": {
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
	"Deployment.apps/default/router": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Router }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='STATS_PASSWORD')].value"),
			Template: "{{ .Config.RouterStatsPassword }}",
		},
	},
	"Deployment.apps/kube-service-catalog/apiserver": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.ServiceCatalog }}",
		},
	},
	"Deployment.apps/kube-service-catalog/controller-manager": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.ServiceCatalog }}",
		},
	},
	"Deployment.apps/openshift-ansible-service-broker/asb": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.AnsibleServiceBroker }}",
		},
	},
	"Deployment.apps/openshift-etcd/etcd-operator": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.EtcdOperator }}",
		},
	},
	"Deployment.apps/openshift-infra/bootstrap-autoapprover": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.Node }}",
		},
	},
	"Deployment.apps/openshift-template-service-broker/apiserver": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.Images.TemplateServiceBroker }}",
		},
	},
	"Deployment.apps/openshift-metrics/kube-state-metrics": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='kube-state-metrics')].image"),
			Template: "{{ .Config.Images.KubeStateMetrics }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='addon-resizer')].image"),
			Template: "{{ .Config.Images.AddonsResizer }}",
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
			Template: "https://registry-console-default.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
		{
			Path:     jsonpath.MustCompile("$.secret"),
			Template: "{{ .Config.RegistryConsoleOAuthSecret }}",
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
	"Route.route.openshift.io/openshift-metrics/alertmanager": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "alertmanager-openshift-metrics.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-metrics/alerts": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "alerts-openshift-metrics.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-metrics/prometheus": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "prometheus-openshift-metrics.{{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain }}",
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
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
				b, err := config.Derived.CloudProviderConf(cs)
				return string(b), err
			},
		},
	},
	"Secret/default/router-certs": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.Router.Cert) }}\n{{ String (CertAsBytes .Config.Certificates.Ca.Cert) }}\n{{ String (PrivateKeyAsBytes .Config.Certificates.Router.Key) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.Router.Key) }}",
		},
	},
	"Secret/kube-service-catalog/apiserver-ssl": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'apiserver.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.ServiceCatalogServer.Cert) }}\n{{ String (CertAsBytes .Config.Certificates.ServiceCatalogCa.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'apiserver.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.ServiceCatalogServer.Key) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'etcd-ca.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.EtcdCa.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'etcd-client.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.EtcdClient.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'etcd-client.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.EtcdClient.Key) }}",
		},
	},
	"Secret/openshift-etcd/etcd-backup-abs-credentials": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'storage-account'"),
			Template: "{{ .Config.ConfigStorageAccount }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'storage-key'"),
			Template: "{{ .Extra.ConfigStorageAccountKey }}",
		},
	},
	"Secret/openshift-etcd/etcd-client-tls": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'etcd-client-ca.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.EtcdCa.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'etcd-client.crt'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.EtcdClient.Cert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'etcd-client.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.Certificates.EtcdClient.Key) }}",
		},
	},
	"Secret/openshift-metrics/alertmanager-proxy": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'session_secret'"),
			Template: "{{ Base64Encode .Config.AlertManagerProxySessionSecret }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'console-cert'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.OpenshiftConsole.Cert) }}",
		},
	},
	"Secret/openshift-metrics/alerts-proxy": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'session_secret'"),
			Template: "{{ Base64Encode .Config.AlertsProxySessionSecret }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'console-cert'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.OpenshiftConsole.Cert) }}",
		},
	},
	"Service/default/router": {
		{
			Path: jsonpath.MustCompile("$.metadata.annotations['service.beta.kubernetes.io/azure-dns-label-name']"),
			F: func(cs *acsapi.OpenShiftManagedCluster) (string, error) {
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
	"Secret/openshift-metrics/prometheus-proxy": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'session_secret'"),
			Template: "{{ Base64Encode .Config.PrometheusProxySessionSecret }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'console-cert'"),
			Template: "{{ String (CertAsBytes .Config.Certificates.OpenshiftConsole.Cert) }}",
		},
	},
	"StatefulSet.apps/openshift-metrics/prometheus": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='prom-proxy')].image"),
			Template: "{{ .Config.Images.OAuthProxy }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='prometheus')].image"),
			Template: "{{ .Config.Images.Prometheus }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alerts-proxy')].image"),
			Template: "{{ .Config.Images.OAuthProxy }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alert-buffer')].image"),
			Template: "{{ .Config.Images.PrometheusAlertBuffer }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alertmanager-proxy')].image"),
			Template: "{{ .Config.Images.OAuthProxy }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alertmanager')].image"),
			Template: "{{ .Config.Images.PrometheusAlertManager }}",
		},
	},
	"StorageClass.storage.k8s.io/azure": {
		{
			Path:     jsonpath.MustCompile("$.parameters.location"),
			Template: "{{ .ContainerService.Location }}",
		},
	},
}
