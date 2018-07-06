package addons

import (
	"encoding/base64"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/jim-minter/azure-helm/pkg/jsonpath"
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
}{
	"APIService.apiregistration.k8s.io/v1beta1.servicecatalog.k8s.io": {
		{
			Path:     jsonpath.MustCompile("$.spec.caBundle"),
			Template: "{{ Base64Encode (CertAsBytes .Config.ServiceCatalogCaCert) }}",
		},
	},
	"ClusterServiceBroker.servicecatalog.k8s.io/ansible-service-broker": {
		{
			Path:     jsonpath.MustCompile("$.spec.caBundle"),
			Template: "{{ Base64Encode (CertAsBytes .Config.ServiceSigningCaCert) }}",
		},
	},
	"ClusterServiceBroker.servicecatalog.k8s.io/template-service-broker": {
		{
			Path:     jsonpath.MustCompile("$.spec.caBundle"),
			Template: "{{ Base64Encode (CertAsBytes .Config.ServiceSigningCaCert) }}",
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
			Template: "{{ String (CertAsBytes .Config.CaCert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.data.'requestheader-client-ca-file'"),
			Template: "{{ String (CertAsBytes .Config.FrontProxyCaCert) }}",
		},
	},
	"ConfigMap/openshift-web-console/webconsole-config": {
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.consolePublicURL"),
			Template:   "https://{{ .Manifest.PublicHostname }}/console/",
		},
		{
			Path:       jsonpath.MustCompile("$.data.'webconsole-config.yaml'"),
			NestedPath: jsonpath.MustCompile("$.clusterInfo.masterPublicURL"),
			Template:   "https://{{ .Manifest.PublicHostname }}",
		},
	},
	"DaemonSet.apps/openshift-azure/tunnel": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.TunnelImage }}",
		},
	},
	"DaemonSet.apps/openshift-metrics/prometheus-node-exporter": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.PrometheusNodeExporterImage }}",
		},
	},
	"DaemonSet.apps/openshift-node/sync": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.NodeImage }}",
		},
	},
	"DaemonSet.apps/openshift-sdn/ovs": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.NodeImage }}",
		},
	},
	"DaemonSet.apps/openshift-sdn/sdn": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.NodeImage }}",
		},
	},
	"Deployment.apps/default/docker-registry": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='REGISTRY_HTTP_SECRET')].value"),
			Template: "{{ Base64Encode .Config.RegistryHTTPSecret }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.RegistryImage }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.initContainers[0].env[?(@.name='REGISTRY_STORAGE_ACCOUNT_NAME')].value"),
			Template: "{{ .Config.RegistryStorageAccount }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.initContainers[0].image"),
			Template: "{{ .Config.AzureCLIImage }}",
		},
	},
	"Deployment.apps/default/registry-console": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='OPENSHIFT_OAUTH_PROVIDER_URL')].value"),
			Template: "https://{{ .Manifest.PublicHostname }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].env[?(@.name='REGISTRY_HOST')].value"),
			Template: "docker-registry-default.{{ .Manifest.RoutingConfigSubdomain }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.RegistryConsoleImage }}",
		},
	},
	"Deployment.apps/default/router": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.RouterImage }}",
		},
	},
	"Deployment.apps/kube-service-catalog/controller-manager": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.ServiceCatalogImage }}",
		},
	},
	"Deployment.apps/openshift-ansible-service-broker/asb": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.AnsibleServiceBrokerImage }}",
		},
	},
	"Deployment.apps/openshift-template-service-broker/apiserver": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.TemplateServiceBrokerImage }}",
		},
	},
	"Deployment.apps/openshift-web-console/webconsole": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[0].image"),
			Template: "{{ .Config.WebConsoleImage }}",
		},
	},
	"OAuthClient.oauth.openshift.io/cockpit-oauth-client": {
		{
			Path:     jsonpath.MustCompile("$.redirectURIs[0]"),
			Template: "https://registry-console-default.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Route.route.openshift.io/default/docker-registry": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "docker-registry-default.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Route.route.openshift.io/default/registry-console": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "registry-console-default.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Route.route.openshift.io/kube-service-catalog/apiserver": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "apiserver-kube-service-catalog.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-ansible-service-broker/asb-1338": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "asb-1338-openshift-ansible-service-broker.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-metrics/alertmanager": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "alertmanager-openshift-metrics.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-metrics/alerts": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "alerts-openshift-metrics.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Route.route.openshift.io/openshift-metrics/prometheus": {
		{
			Path:     jsonpath.MustCompile("$.spec.host"),
			Template: "prometheus-openshift-metrics.{{ .Manifest.RoutingConfigSubdomain }}",
		},
	},
	"Secret/default/registry-certificates": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'registry.crt'"),
			Template: "{{ String (CertAsBytes .Config.RegistryCert) }}\n{{ String (CertAsBytes .Config.CaCert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'registry.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.RegistryKey) }}",
		},
	},
	"Secret/default/registry-config": {
		{
			Path:       jsonpath.MustCompile("$.stringData.'config.yml'"),
			NestedPath: jsonpath.MustCompile("$.storage.azure.accountname"),
			Template:   "{{ .Config.RegistryStorageAccount }}",
		},
	},
	"Secret/default/etc-origin-cloudprovider": {
		{
			Path:       jsonpath.MustCompile("$.stringData.'azure.conf'"),
			NestedPath: jsonpath.MustCompile("$.tenantId"),
			Template:   "{{ .Manifest.TenantID }}",
		},
		{
			Path:       jsonpath.MustCompile("$.stringData.'azure.conf'"),
			NestedPath: jsonpath.MustCompile("$.subscriptionId"),
			Template:   "{{ .Manifest.SubscriptionID }}",
		},
		{
			Path:       jsonpath.MustCompile("$.stringData.'azure.conf'"),
			NestedPath: jsonpath.MustCompile("$.aadClientId"),
			Template:   "{{ .Manifest.ClientID }}",
		},
		{
			Path:       jsonpath.MustCompile("$.stringData.'azure.conf'"),
			NestedPath: jsonpath.MustCompile("$.aadClientSecret"),
			Template:   "{{ .Manifest.ClientSecret }}",
		},
		{
			Path:       jsonpath.MustCompile("$.stringData.'azure.conf'"),
			NestedPath: jsonpath.MustCompile("$.aadTenantId"),
			Template:   "{{ .Manifest.TenantID }}",
		},
		{
			Path:       jsonpath.MustCompile("$.stringData.'azure.conf'"),
			NestedPath: jsonpath.MustCompile("$.resourceGroup"),
			Template:   "{{ .Manifest.ResourceGroup }}",
		},
		{
			Path:       jsonpath.MustCompile("$.stringData.'azure.conf'"),
			NestedPath: jsonpath.MustCompile("$.location"),
			Template:   "{{ .Manifest.Location }}",
		},
	},
	"Secret/default/router-certs": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.crt'"),
			Template: "{{ String (CertAsBytes .Config.RouterCert) }}\n{{ String (CertAsBytes .Config.CaCert) }}\n{{ String (PrivateKeyAsBytes .Config.RouterKey) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.RouterKey) }}",
		},
	},
	"Secret/kube-service-catalog/apiserver-ssl": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.crt'"),
			Template: "{{ String (CertAsBytes .Config.ServiceCatalogServerCert) }}\n{{ String (CertAsBytes .Config.ServiceCatalogCaCert) }}",
		},
		{
			Path:     jsonpath.MustCompile("$.stringData.'tls.key'"),
			Template: "{{ String (PrivateKeyAsBytes .Config.ServiceCatalogServerKey) }}",
		},
	},
	"Secret/openshift-azure/tunnel": {
		{
			Path:       jsonpath.MustCompile("$.stringData.'tunnel.conf'"),
			NestedPath: jsonpath.MustCompile("$.address"),
			Template:   "{{ .Config.TunnelHostname }}:443",
		},
	},
	"Secret/openshift-metrics/alertmanager-proxy": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'session_secret'"),
			Template: "{{ Base64Encode .Config.AlertManagerProxySessionSecret }}",
		},
	},
	"Secret/openshift-metrics/alerts-proxy": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'session_secret'"),
			Template: "{{ Base64Encode .Config.AlertsProxySessionSecret }}",
		},
	},
	"Secret/openshift-metrics/prometheus-proxy": {
		{
			Path:     jsonpath.MustCompile("$.stringData.'session_secret'"),
			Template: "{{ Base64Encode .Config.PrometheusProxySessionSecret }}",
		},
	},
	"StatefulSet.apps/openshift-metrics/prometheus": {
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='prom-proxy')].image"),
			Template: "{{ .Config.OAuthProxyImage }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='prometheus')].image"),
			Template: "{{ .Config.PrometheusImage }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alerts-proxy')].image"),
			Template: "{{ .Config.OAuthProxyImage }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alert-buffer')].image"),
			Template: "{{ .Config.PrometheusAlertBufferImage }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alertmanager-proxy')].image"),
			Template: "{{ .Config.OAuthProxyImage }}",
		},
		{
			Path:     jsonpath.MustCompile("$.spec.template.spec.containers[?(@.name='alertmanager')].image"),
			Template: "{{ .Config.PrometheusAlertManagerImage }}",
		},
	},
}
