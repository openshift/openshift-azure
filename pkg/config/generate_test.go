package config

import (
	"reflect"
	"testing"

	"github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func TestGenerate(t *testing.T) {
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			OpenShiftVersion: "v3.11",
			RouterProfiles: []api.RouterProfile{
				{},
			},
		},
	}
	pc := api.PluginConfig{
		TestConfig: api.TestConfig{
			RunningUnderTest: true,
		},
	}

	prepare := func(v reflect.Value) {}
	var template *pluginapi.Config
	populate.Walk(&template, prepare)

	cg := simpleGenerator{pluginConfig: pc}
	err := cg.Generate(cs, template)
	if err != nil {
		t.Error(err)
	}

	testRequiredFields(cs, t)
}

func testRequiredFields(cs *api.OpenShiftManagedCluster, t *testing.T) {
	assert := func(c bool, name string) {
		if !c {
			t.Errorf("missing %s", name)
		}
	}
	assertCert := func(c api.CertKeyPair, name string) {
		assert(c.Key != nil, name+" key")
		assert(c.Cert != nil, name+" cert")
	}

	c := cs.Config

	assert(c.ImagePublisher != "", "image publisher")
	assert(c.ImageOffer != "", "image offer")
	assert(c.ImageVersion != "", "image version")

	assert(c.Images.Format != "", "image config format")
	assert(c.Images.ControlPlane != "", "control plane image")
	assert(c.Images.Node != "", "node image")
	assert(c.Images.ServiceCatalog != "", "service catalog image")
	assert(c.Images.AnsibleServiceBroker != "", "ansible service broker image")
	assert(c.Images.TemplateServiceBroker != "", "template service broker image")
	assert(c.Images.Registry != "", "registry image")
	assert(c.Images.Router != "", "router image")
	assert(c.Images.WebConsole != "", "web console image")
	assert(c.Images.Console != "", "console image")
	assert(c.Images.MasterEtcd != "", "master etcd image")
	assert(c.Images.RegistryConsole != "", "registry console image")
	assert(c.Images.Sync != "", "sync image")
	assert(c.Images.Startup != "", "startup image")
	assert(c.Images.EtcdBackup != "", "etcdbackup image")
	assert(c.Images.ClusterMonitoringOperator != "", "cluster monitoring operator image")
	assert(c.Images.PrometheusOperatorBase != "", "cluster monitoring operator image")
	assert(c.Images.PrometheusConfigReloaderBase != "", "prometheus config reloader base image")
	assert(c.Images.ConfigReloaderBase != "", "config reloader base image")
	assert(c.Images.PrometheusBase != "", "prometheus base image")
	assert(c.Images.AlertManagerBase != "", "alertmanager base image")
	assert(c.Images.NodeExporterBase != "", "node exporter base image")
	assert(c.Images.GrafanaBase != "", "grafana base image")
	assert(c.Images.KubeStateMetricsBase != "", "kube state metrics base image")
	assert(c.Images.KubeRbacProxyBase != "", "kube rbac proxy base image")
	assert(c.Images.OAuthProxyBase != "", "oauth proxy base image")
	assert(c.Images.GenevaLogging != "", "azure logging image")
	assert(c.Images.GenevaTDAgent != "", "azure TDAgent image")
	assert(c.Images.MetricsBridge != "", "metrics-bridge image")

	assert(c.ServiceAccountKey != nil, "service account key")

	assert(c.SSHKey != nil, "ssh key")

	assert(len(c.RegistryStorageAccount) != 0, "registry storage account")
	assert(len(c.RegistryConsoleOAuthSecret) != 0, "registry console oauth secret")
	assert(len(c.RouterStatsPassword) != 0, "router stats password")

	assert(c.ServiceCatalogClusterID != uuid.Nil, "service catalog cluster id")

	assertCert(c.Certificates.EtcdCa, "EtcdCa")
	assertCert(c.Certificates.Ca, "Ca")
	assertCert(c.Certificates.FrontProxyCa, "FrontProxyCa")
	assertCert(c.Certificates.ServiceSigningCa, "ServiceSigningCa")
	assertCert(c.Certificates.ServiceCatalogCa, "ServiceCatalogCa")
	assertCert(c.Certificates.EtcdServer, "EtcdServer")
	assertCert(c.Certificates.EtcdPeer, "EtcdPeer")
	assertCert(c.Certificates.EtcdClient, "EtcdClient")
	assertCert(c.Certificates.MasterServer, "MasterServer")
	assertCert(c.Certificates.Admin, "Admin")
	assertCert(c.Certificates.AggregatorFrontProxy, "AggregatorFrontProxy")
	assertCert(c.Certificates.MasterKubeletClient, "MasterKubeletClient")
	assertCert(c.Certificates.MasterProxyClient, "MasterProxyClient")
	assertCert(c.Certificates.OpenShiftMaster, "OpenShiftMaster")
	assertCert(c.Certificates.NodeBootstrap, "NodeBootstrap")
	assertCert(c.Certificates.Registry, "Registry")
	assertCert(c.Certificates.ServiceCatalogServer, "ServiceCatalogServer")
	assertCert(c.Certificates.AzureClusterReader, "AzureClusterReader")

	assert(len(c.SessionSecretAuth) != 0, "SessionSecretAuth")
	assert(len(c.SessionSecretEnc) != 0, "SessionSecretEnc")
	assert(len(c.RegistryHTTPSecret) != 0, "RegistryHTTPSecret")
	assert(len(c.AlertManagerProxySessionSecret) != 0, "AlertManagerProxySessionSecret")
	assert(len(c.AlertsProxySessionSecret) != 0, "AlertsProxySessionSecret")
	assert(len(c.PrometheusProxySessionSecret) != 0, "PrometheusProxySessionSecret")

	assert(c.MasterKubeconfig != nil, "MasterKubeconfig")
	assert(c.AdminKubeconfig != nil, "AdminKubeconfig")
	assert(c.NodeBootstrapKubeconfig != nil, "NodeBootstrapKubeconfig")
	assert(c.AzureClusterReaderKubeconfig != nil, "AzureClusterReaderKubeconfig")
}

func TestInvalidateSecrets(t *testing.T) {
	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []api.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]api.IdentityProvider{{Provider: &api.AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}
	cs := &api.OpenShiftManagedCluster{}
	template := &pluginapi.Config{}
	populate.Walk(cs, prepare)
	populate.Walk(template, prepare)

	var g simpleGenerator
	saved := cs.DeepCopy()
	if err := g.InvalidateSecrets(cs); err != nil {
		t.Errorf("configGenerator.InvalidateSecrets error = %v", err)
	}
	g.Generate(cs, template)

	// compare fields that are expected to be different
	if reflect.DeepEqual(saved.Config.Certificates.Admin, cs.Config.Certificates.Admin) {
		t.Errorf("expected change to Admin certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.AggregatorFrontProxy, cs.Config.Certificates.AggregatorFrontProxy) {
		t.Errorf("expected change to AggregatorFrontProxy certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.AzureClusterReader, cs.Config.Certificates.AzureClusterReader) {
		t.Errorf("expected change to AzureClusterReader certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.EtcdClient, cs.Config.Certificates.EtcdClient) {
		t.Errorf("expected change to EtcdClient certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.EtcdPeer, cs.Config.Certificates.EtcdPeer) {
		t.Errorf("expected change to EtcdPeer certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.EtcdServer, cs.Config.Certificates.EtcdServer) {
		t.Errorf("expected change to EtcdServer certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.MasterKubeletClient, cs.Config.Certificates.MasterKubeletClient) {
		t.Errorf("expected change to MasterKubeletClient certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.MasterProxyClient, cs.Config.Certificates.MasterProxyClient) {
		t.Errorf("expected change to MasterProxyClient certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.MasterServer, cs.Config.Certificates.MasterServer) {
		t.Errorf("expected change to MasterProxyClient certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.NodeBootstrap, cs.Config.Certificates.NodeBootstrap) {
		t.Errorf("expected change to NodeBootstrap certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.OpenShiftMaster, cs.Config.Certificates.OpenShiftMaster) {
		t.Errorf("expected change to OpenShiftMaster certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.Registry, cs.Config.Certificates.Registry) {
		t.Errorf("expected change to Registry certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Certificates.ServiceCatalogServer, cs.Config.Certificates.ServiceCatalogServer) {
		t.Errorf("expected change to ServiceCatalogServer certificates after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.SSHKey, cs.Config.SSHKey) {
		t.Errorf("expected change to SSHKey after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.RegistryHTTPSecret, cs.Config.RegistryHTTPSecret) {
		t.Errorf("expected change to RegistryHTTPSecret after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.RegistryConsoleOAuthSecret, cs.Config.RegistryConsoleOAuthSecret) {
		t.Errorf("expected change to RegistryConsoleOAuthSecret after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.ConsoleOAuthSecret, cs.Config.ConsoleOAuthSecret) {
		t.Errorf("expected change to ConsoleOAuthSecret after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.AlertManagerProxySessionSecret, cs.Config.AlertManagerProxySessionSecret) {
		t.Errorf("expected change to AlertManagerProxySessionSecret after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.AlertsProxySessionSecret, cs.Config.AlertsProxySessionSecret) {
		t.Errorf("expected change to AlertsProxySessionSecret after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.PrometheusProxySessionSecret, cs.Config.PrometheusProxySessionSecret) {
		t.Errorf("expected change to PrometheusProxySessionSecret after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.SessionSecretAuth, cs.Config.SessionSecretAuth) {
		t.Errorf("expected change to SessionSecretAuth after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.SessionSecretEnc, cs.Config.SessionSecretEnc) {
		t.Errorf("expected change to SessionSecretEnc after secret invalidation")
	}
	if reflect.DeepEqual(saved.Config.Images.GenevaImagePullSecret, cs.Config.Images.GenevaImagePullSecret) {
		t.Errorf("expected change to GenevaImagePullSecret after secret invalidation")
	}

	// compare fields that are expected to be the same
	if !reflect.DeepEqual(saved.Config.Certificates.GenevaMetrics, cs.Config.Certificates.GenevaMetrics) {
		t.Errorf("unexpected change to GenevaMetrics certificates after secret invalidation")
	}
	if !reflect.DeepEqual(saved.Config.Certificates.GenevaLogging, cs.Config.Certificates.GenevaLogging) {
		t.Errorf("unexpected change to GenevaLogging after secret invalidation")
	}

	// assign saved values back to those changed in cs
	cs.Config.Certificates.Admin = saved.Config.Certificates.Admin
	cs.Config.Certificates.AggregatorFrontProxy = saved.Config.Certificates.AggregatorFrontProxy
	cs.Config.Certificates.AzureClusterReader = saved.Config.Certificates.AzureClusterReader
	cs.Config.Certificates.EtcdClient = saved.Config.Certificates.EtcdClient
	cs.Config.Certificates.EtcdPeer = saved.Config.Certificates.EtcdPeer
	cs.Config.Certificates.EtcdServer = saved.Config.Certificates.EtcdServer
	cs.Config.Certificates.MasterKubeletClient = saved.Config.Certificates.MasterKubeletClient
	cs.Config.Certificates.MasterProxyClient = saved.Config.Certificates.MasterProxyClient
	cs.Config.Certificates.MasterServer = saved.Config.Certificates.MasterServer
	cs.Config.Certificates.NodeBootstrap = saved.Config.Certificates.NodeBootstrap
	cs.Config.Certificates.OpenShiftMaster = saved.Config.Certificates.OpenShiftMaster
	cs.Config.Certificates.Registry = saved.Config.Certificates.Registry
	cs.Config.Certificates.ServiceCatalogServer = saved.Config.Certificates.ServiceCatalogServer
	cs.Config.Certificates.GenevaLogging = saved.Config.Certificates.GenevaLogging
	cs.Config.Certificates.GenevaMetrics = saved.Config.Certificates.GenevaMetrics

	// compare certs from saved and cs
	if !reflect.DeepEqual(saved.Config.Certificates, cs.Config.Certificates) {
		t.Errorf("expected saved and cs config blobs to be equal")
	}
}
