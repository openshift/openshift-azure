package config

import (
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
)

var testOpenShiftClusterYAML = []byte(`---
id: openshift
location: eastus
name: test-cluster
config:
  version: 310
properties:
  fqdn: "console-internal.example.com"
  publicHostname: ""
  routerProfiles:
  - fqdn: router-internal.example.com
    name: router
    publicSubdomain: ""
`)

func TestGenerate(t *testing.T) {
	tests := map[string]struct {
		omc *api.OpenShiftManagedCluster
	}{
		"test generate new": {
			omc: fixtures.NewTestOpenShiftCluster(),
		},
		//TODO "test generate doesn't overwrite": {},
	}

	for name, test := range tests {
		err := Generate(test.omc)
		if err != nil {
			t.Errorf("%s received generation error %v", name, err)
			continue
		}
		testRequiredFields(test.omc, t)
		// check mutation
	}
}

func testRequiredFields(omc *api.OpenShiftManagedCluster, t *testing.T) {
	assert := func(c bool, name string) {
		if !c {
			t.Errorf("missing %s", name)
		}
	}
	assertCert := func(c api.CertKeyPair, name string) {
		assert(c.Key != nil, name+" key")
		assert(c.Cert != nil, name+" cert")
	}

	c := omc.Config
	assert(c.ImagePublisher != "", "image publisher")
	assert(c.ImageOffer != "", "image offer")
	assert(c.ImageVersion != "", "image version")
	assert(c.ImageConfigFormat != "", "image config format")

	assert(c.ControlPlaneImage != "", "control plane image")
	assert(c.NodeImage != "", "node image")
	assert(c.ServiceCatalogImage != "", "service catalog image")
	assert(c.AnsibleServiceBrokerImage != "", "ansible service broker image")
	assert(c.TemplateServiceBrokerImage != "", "template service broker image")
	assert(c.RegistryImage != "", "registry image")
	assert(c.RouterImage != "", "router image")
	assert(c.WebConsoleImage != "", "web console image")
	assert(c.MasterEtcdImage != "", "master etcd image")
	assert(c.OAuthProxyImage != "", "oauth proxy image")
	assert(c.PrometheusImage != "", "prometheus image")
	assert(c.PrometheusAlertBufferImage != "", "alert buffer image")
	assert(c.PrometheusAlertManagerImage != "", "alert manager image")
	assert(c.PrometheusNodeExporterImage != "", "node exporter image")
	assert(c.RegistryConsoleImage != "", "registry console image")
	assert(c.AzureCLIImage != "", "azure cli image")
	assert(c.SyncImage != "", "sync image")
	assert(c.LogBridgeImage != "", "logbridge image")

	assert(omc.Properties.PublicHostname != "", "public host name")
	assert(omc.Properties.RouterProfiles[0].PublicSubdomain != "", "router public subdomain")

	assert(c.ServiceAccountKey != nil, "service account key")
	assert(len(c.HtPasswd) != 0, "htpassword")
	assert(c.SSHKey != nil, "ssh key")

	assert(len(c.RegistryStorageAccount) != 0, "registry storage account")
	assert(len(c.RegistryConsoleOAuthSecret) != 0, "registry console oauth secret")
	assert(len(c.RouterStatsPassword) != 0, "router stats password")
	assert(len(c.LoggingWorkspace) != 0, "logging workspace")

	assert(c.ServiceCatalogClusterID.UUID != uuid.Nil, "service catalog cluster id")

	assert(c.TenantID != "", "tenant id")
	assert(c.SubscriptionID != "", "subscription id")
	assert(c.ResourceGroup != "", "resource group")

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

func TestSelectDNSNames(t *testing.T) {
	tests := map[string]struct {
		f        func(*api.OpenShiftManagedCluster)
		expected func(*api.OpenShiftManagedCluster)
	}{
		"test no PublicHostname": {
			f: func(cs *api.OpenShiftManagedCluster) {},
			expected: func(cs *api.OpenShiftManagedCluster) {
				cs.Properties.PublicHostname = "console-internal.example.com"
				cs.Properties.RouterProfiles[0].PublicSubdomain = "router-internal.example.com"
				cs.Config.RouterLBCNamePrefix = "router-internal"
				cs.Config.MasterLBCNamePrefix = "console-internal"
			},
		},
		"test no PublicHostname for router": {
			f: func(cs *api.OpenShiftManagedCluster) {
				cs.Properties.PublicHostname = "console.example.com"
			},
			expected: func(cs *api.OpenShiftManagedCluster) {
				cs.Properties.RouterProfiles[0].PublicSubdomain = "router-internal.example.com"
				cs.Properties.PublicHostname = "console.example.com"
				cs.Config.MasterLBCNamePrefix = "console-internal"
				cs.Config.RouterLBCNamePrefix = "router-internal"
			},
		},
		"test master & router prefix configuration": {
			f: func(cs *api.OpenShiftManagedCluster) {
				cs.Properties.RouterProfiles[0].FQDN = "router-custom.test.com"
				cs.Properties.FQDN = "master-custom.test.com"
			},
			expected: func(cs *api.OpenShiftManagedCluster) {
				cs.Properties.RouterProfiles[0].FQDN = "router-custom.test.com"
				cs.Properties.FQDN = "master-custom.test.com"
				cs.Config.MasterLBCNamePrefix = "master-custom"
				cs.Config.RouterLBCNamePrefix = "router-custom"
				cs.Properties.RouterProfiles[0].PublicSubdomain = "router-custom.test.com"
				cs.Properties.PublicHostname = "master-custom.test.com"
			},
		},
	}

	for name, test := range tests {
		input := new(api.OpenShiftManagedCluster)
		output := new(api.OpenShiftManagedCluster)
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &input)
		if err != nil {
			t.Fatal(err)
		}
		// TODO: This can be replaced by deepCopy of input before we populate it
		err = yaml.Unmarshal(testOpenShiftClusterYAML, &output)
		if err != nil {
			t.Fatal(err)
		}

		if test.f != nil {
			test.f(input)
		}
		if test.expected != nil {
			test.expected(output)
		}

		selectDNSNames(input)

		if !reflect.DeepEqual(input, output) {
			t.Errorf("%v: SelectDNSNames test returned unexpected result \n %#v != %#v", name, input, output)
		}

	}
}
