package config

import (
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"

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

	prepare := func(v reflect.Value) {}
	var template *pluginapi.Config
	populate.Walk(&template, prepare)

	cg := simpleGenerator{runningUnderTest: true}
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
	assert(c.Images.Httpd != "", "httpd image")
	assert(c.Images.ClusterMonitoringOperator != "", "cluster monitoring operator image")
	assert(c.Images.PrometheusOperator != "", "cluster monitoring operator image")
	assert(c.Images.PrometheusConfigReloader != "", "prometheus config reloader image")
	assert(c.Images.ConfigReloader != "", "config reloader image")
	assert(c.Images.Prometheus != "", "prometheus image")
	assert(c.Images.AlertManager != "", "alertmanager image")
	assert(c.Images.NodeExporter != "", "node exporter image")
	assert(c.Images.Grafana != "", "grafana image")
	assert(c.Images.KubeStateMetrics != "", "kube state metrics image")
	assert(c.Images.KubeRbacProxy != "", "kube rbac proxy image")
	assert(c.Images.OAuthProxy != "", "oauth proxy image")
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
	assertCert(c.Certificates.RegistryConsole, "RegistryConsole")
	assertCert(c.Certificates.ServiceCatalogServer, "ServiceCatalogServer")
	assertCert(c.Certificates.BlackBoxMonitor, "BlackBoxMonitor")

	assert(len(c.SessionSecretAuth) != 0, "SessionSecretAuth")
	assert(len(c.SessionSecretEnc) != 0, "SessionSecretEnc")
	assert(len(c.RegistryHTTPSecret) != 0, "RegistryHTTPSecret")
	assert(len(c.AlertManagerProxySessionSecret) != 0, "AlertManagerProxySessionSecret")
	assert(len(c.AlertsProxySessionSecret) != 0, "AlertsProxySessionSecret")
	assert(len(c.PrometheusProxySessionSecret) != 0, "PrometheusProxySessionSecret")

	assert(c.MasterKubeconfig != nil, "MasterKubeconfig")
	assert(c.AdminKubeconfig != nil, "AdminKubeconfig")
	assert(c.NodeBootstrapKubeconfig != nil, "NodeBootstrapKubeconfig")
	assert(c.BlackBoxMonitorKubeconfig != nil, "BlackBoxMonitorKubeconfig")
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
	if reflect.DeepEqual(saved.Config.Certificates.BlackBoxMonitor, cs.Config.Certificates.BlackBoxMonitor) {
		t.Errorf("expected change to BlackBoxMonitor certificates after secret invalidation")
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
	if reflect.DeepEqual(saved.Config.Certificates.RegistryConsole, cs.Config.Certificates.RegistryConsole) {
		t.Errorf("expected change to RegistryConsole certificates after secret invalidation")
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
	cs.Config.Certificates.BlackBoxMonitor = saved.Config.Certificates.BlackBoxMonitor
	cs.Config.Certificates.EtcdClient = saved.Config.Certificates.EtcdClient
	cs.Config.Certificates.EtcdPeer = saved.Config.Certificates.EtcdPeer
	cs.Config.Certificates.EtcdServer = saved.Config.Certificates.EtcdServer
	cs.Config.Certificates.MasterKubeletClient = saved.Config.Certificates.MasterKubeletClient
	cs.Config.Certificates.MasterProxyClient = saved.Config.Certificates.MasterProxyClient
	cs.Config.Certificates.MasterServer = saved.Config.Certificates.MasterServer
	cs.Config.Certificates.NodeBootstrap = saved.Config.Certificates.NodeBootstrap
	cs.Config.Certificates.OpenShiftMaster = saved.Config.Certificates.OpenShiftMaster
	cs.Config.Certificates.Registry = saved.Config.Certificates.Registry
	cs.Config.Certificates.RegistryConsole = saved.Config.Certificates.RegistryConsole
	cs.Config.Certificates.ServiceCatalogServer = saved.Config.Certificates.ServiceCatalogServer
	cs.Config.Certificates.GenevaLogging = saved.Config.Certificates.GenevaLogging
	cs.Config.Certificates.GenevaMetrics = saved.Config.Certificates.GenevaMetrics

	// compare certs from saved and cs
	if !reflect.DeepEqual(saved.Config.Certificates, cs.Config.Certificates) {
		t.Errorf("expected saved and cs config blobs to be equal")
	}
}
