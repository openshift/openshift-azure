package config

import (
	"testing"

	"github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
)

func TestGenerate(t *testing.T) {
	tests := map[string]struct {
		cs *api.OpenShiftManagedCluster
	}{
		"test generate new": {
			cs: fixtures.NewTestOpenShiftCluster(),
		},
		//TODO "test generate doesn't overwrite": {},
	}

	for name, test := range tests {
		err := Generate(test.cs)
		if err != nil {
			t.Errorf("%s received generation error %v", name, err)
			continue
		}
		testRequiredFields(test.cs, t)
		// check mutation
	}
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
	assert(c.Images.MasterEtcd != "", "master etcd image")
	assert(c.Images.OAuthProxy != "", "oauth proxy image")
	assert(c.Images.Prometheus != "", "prometheus image")
	assert(c.Images.PrometheusAlertBuffer != "", "alert buffer image")
	assert(c.Images.PrometheusAlertManager != "", "alert manager image")
	assert(c.Images.PrometheusNodeExporter != "", "node exporter image")
	assert(c.Images.RegistryConsole != "", "registry console image")
	assert(c.Images.Sync != "", "sync image")
	assert(c.Images.LogBridge != "", "logbridge image")

	assert(c.ServiceAccountKey != nil, "service account key")
	assert(len(c.HtPasswd) != 0, "htpassword")
	assert(len(c.AdminPasswd) != 0, "admin password")
	assert(c.SSHKey != nil, "ssh key")

	assert(len(c.RegistryStorageAccount) != 0, "registry storage account")
	assert(len(c.RegistryConsoleOAuthSecret) != 0, "registry console oauth secret")
	assert(len(c.RouterStatsPassword) != 0, "router stats password")
	assert(len(c.LoggingWorkspace) != 0, "logging workspace")

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
	assertCert(c.Certificates.OpenshiftConsole, "OpenshiftConsole")
	assertCert(c.Certificates.Admin, "Admin")
	assertCert(c.Certificates.AggregatorFrontProxy, "AggregatorFrontProxy")
	assertCert(c.Certificates.MasterKubeletClient, "MasterKubeletClient")
	assertCert(c.Certificates.MasterProxyClient, "MasterProxyClient")
	assertCert(c.Certificates.OpenShiftMaster, "OpenShiftMaster")
	assertCert(c.Certificates.NodeBootstrap, "NodeBootstrap")
	assertCert(c.Certificates.Registry, "Registry")
	assertCert(c.Certificates.Router, "Router")
	assertCert(c.Certificates.ServiceCatalogServer, "ServiceCatalogServer")
	assertCert(c.Certificates.ServiceCatalogAPIClient, "ServiceCatalogAPIClient")
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
