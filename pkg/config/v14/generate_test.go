package config

import (
	"bytes"
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/util/cmp"
	"github.com/openshift/openshift-azure/test/util/populate"
	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestGenerate(t *testing.T) {
	cs := &api.OpenShiftManagedCluster{
		Config: api.Config{
			PluginVersion: "Versions.key",
		},
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

	cg := simpleGenerator{cs: cs}
	err := cg.Generate(template, true)
	if err != nil {
		t.Fatal(err)
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
	assert(c.Images.TLSProxy != "", "tlsProxy image")
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
	assert(c.Images.Canary != "", "canary image")
	assert(c.Images.AroAdmissionController != "", "aro admission controller image")
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
	assert(c.Images.MetricsServer != "", "metrics-server image")

	assert(c.ServiceAccountKey != nil, "service account key")

	assert(c.SSHKey != nil, "ssh key")

	assert(len(c.RegistryStorageAccount) != 0, "registry storage account")
	assert(len(c.RegistryConsoleOAuthSecret) != 0, "registry console oauth secret")
	assert(len(c.RouterStatsPassword) != 0, "router stats password")
	assert(len(c.EtcdMetricsPassword) != 0, "etcd metrics password")
	assert(len(c.EtcdMetricsUsername) != 0, "etcd metrics username")

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
	assertCert(c.Certificates.SDN, "SDN")
	assertCert(c.Certificates.Registry, "Registry")
	assertCert(c.Certificates.RegistryConsole, "RegistryConsole")
	assertCert(c.Certificates.ServiceCatalogServer, "ServiceCatalogServer")
	assertCert(c.Certificates.AroAdmissionController, "AroAdmissionController")
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
	assert(c.SDNKubeconfig != nil, "SDNKubeconfig")
	assert(c.BlackBoxMonitorKubeconfig != nil, "BlackBoxMonitorKubeconfig")
}

func TestInvalidateSecrets(t *testing.T) {
	assert := func(c bool, name string) {
		if !c {
			t.Errorf("%s same", name)
		}
	}

	cs := &api.OpenShiftManagedCluster{}
	pluginConfig := &pluginapi.Config{}

	g := simpleGenerator{cs: cs}
	err := g.Generate(pluginConfig, false)
	if err != nil {
		t.Error(err)
	}

	saved := cs.DeepCopy()

	err = g.InvalidateSecrets()
	if err != nil {
		t.Error(err)
	}

	pluginConfig.GenevaImagePullSecret = []byte("genevaImagePullSecret")
	pluginConfig.ImagePullSecret = []byte("imagePullSecret")
	pluginConfig.Certificates.GenevaLogging.Cert = tls.DummyCertificate
	pluginConfig.Certificates.GenevaMetrics.Cert = tls.DummyCertificate
	pluginConfig.Certificates.PackageRepository.Cert = tls.DummyCertificate

	err = g.Generate(pluginConfig, false)
	if err != nil {
		t.Error(err)
	}

	// compare fields that are expected to be different
	assert(!reflect.DeepEqual(cs.Config.SSHKey, saved.Config.SSHKey), "sshKey")
	assert(!reflect.DeepEqual(cs.Config.Certificates.GenevaLogging, saved.Config.Certificates.GenevaLogging), "certificates.genevaLogging")
	assert(!reflect.DeepEqual(cs.Config.Certificates.GenevaMetrics, saved.Config.Certificates.GenevaMetrics), "certificates.genevaMetrics")
	assert(!reflect.DeepEqual(cs.Config.Certificates.PackageRepository, saved.Config.Certificates.PackageRepository), "certificates.packageRepository")
	assert(!bytes.Equal(cs.Config.Images.GenevaImagePullSecret, saved.Config.Images.GenevaImagePullSecret), "images.genevaImagePullSecret")
	assert(!bytes.Equal(cs.Config.Images.ImagePullSecret, saved.Config.Images.ImagePullSecret), "images.imagePullSecret")
	assert(!bytes.Equal(cs.Config.SessionSecretAuth, saved.Config.SessionSecretAuth), "sessionSecretAuth")
	assert(!bytes.Equal(cs.Config.SessionSecretEnc, saved.Config.SessionSecretEnc), "sessionSecreteEnc")
	assert(!bytes.Equal(cs.Config.RegistryHTTPSecret, saved.Config.RegistryHTTPSecret), "registryHTTPSecret")
	assert(!bytes.Equal(cs.Config.PrometheusProxySessionSecret, saved.Config.PrometheusProxySessionSecret), "prometheusProxySessionSecret")
	assert(!bytes.Equal(cs.Config.AlertManagerProxySessionSecret, saved.Config.AlertManagerProxySessionSecret), "alertManagerProxySessionSecret")
	assert(!bytes.Equal(cs.Config.AlertsProxySessionSecret, saved.Config.AlertsProxySessionSecret), "alertsProxySessionSecret")
	assert(cs.Config.RegistryConsoleOAuthSecret != saved.Config.RegistryConsoleOAuthSecret, "registryConsoleOAuthSecret")
	assert(cs.Config.ConsoleOAuthSecret != saved.Config.ConsoleOAuthSecret, "consoleOAuthSecret")
	assert(cs.Config.RouterStatsPassword != saved.Config.RouterStatsPassword, "routerStatsPassword")
	assert(cs.Config.EtcdMetricsPassword != saved.Config.EtcdMetricsPassword, "etcdMetricsPassword")
	assert(cs.Config.EtcdMetricsUsername != saved.Config.EtcdMetricsUsername, "etcdMetricsUsername")

	// assign saved values back to those changed in cs
	cs.Config.SSHKey = saved.Config.SSHKey
	cs.Config.Certificates.GenevaLogging = saved.Config.Certificates.GenevaLogging
	cs.Config.Certificates.GenevaMetrics = saved.Config.Certificates.GenevaMetrics
	cs.Config.Certificates.PackageRepository = saved.Config.Certificates.PackageRepository
	cs.Config.Images.GenevaImagePullSecret = saved.Config.Images.GenevaImagePullSecret
	cs.Config.Images.ImagePullSecret = saved.Config.Images.ImagePullSecret
	cs.Config.SessionSecretAuth = saved.Config.SessionSecretAuth
	cs.Config.SessionSecretEnc = saved.Config.SessionSecretEnc
	cs.Config.RegistryHTTPSecret = saved.Config.RegistryHTTPSecret
	cs.Config.PrometheusProxySessionSecret = saved.Config.PrometheusProxySessionSecret
	cs.Config.AlertManagerProxySessionSecret = saved.Config.AlertManagerProxySessionSecret
	cs.Config.AlertsProxySessionSecret = saved.Config.AlertsProxySessionSecret
	cs.Config.RegistryConsoleOAuthSecret = saved.Config.RegistryConsoleOAuthSecret
	cs.Config.ConsoleOAuthSecret = saved.Config.ConsoleOAuthSecret
	cs.Config.RouterStatsPassword = saved.Config.RouterStatsPassword
	cs.Config.EtcdMetricsPassword = saved.Config.EtcdMetricsPassword
	cs.Config.EtcdMetricsUsername = saved.Config.EtcdMetricsUsername

	// compare certs from saved and cs
	if !reflect.DeepEqual(cs, saved) {
		t.Errorf("expected saved and cs config blobs to be equal: %s", cmp.Diff(cs, saved))
	}
}
