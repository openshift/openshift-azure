package config

import (
	"crypto/rsa"
	"crypto/x509"
	"net"
	"testing"

	"github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/tls"
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
	var pluginConfig api.PluginConfig
	for name, test := range tests {
		err := Generate(test.cs, pluginConfig)
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
	assert(len(c.LoggingLocation) != 0, "logging location")

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

func TestNeedsGenerate(t *testing.T) {
	var certPlaceholder *x509.Certificate
	var keyPlaceholder *rsa.PrivateKey
	// generate signing cert for certificate
	signingKey, signingCert, err := tls.NewCA("test-ca")
	if err != nil {
		t.Fatal(err)
	}

	// construct certificate test object
	cert := certificate{
		cn:           "test-cn",
		organization: []string{"test-corp"},
		dnsNames: []string{
			"hostname1",
			"hostname2",
		},
		ipAddresses: []net.IP{net.ParseIP("192.168.0.1")},
		extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		signingKey:  signingKey,
		signingCert: signingCert,
		cert:        &certPlaceholder,
		key:         &keyPlaceholder,
	}

	// finish certificate test object with generated cert values.
	*cert.key, *cert.cert, err = tls.NewCert(cert.cn, cert.organization, cert.dnsNames, cert.ipAddresses, cert.extKeyUsage, cert.signingKey, cert.signingCert, false)
	if err != nil {
		t.Fatal(err)
	}

	// not to contaminate cert
	tests := map[string]struct {
		f              func(certificate) certificate
		expectedResult bool
	}{
		"no changes": {
			f: func(cert certificate) certificate {
				return cert
			},
			expectedResult: false,
		},
		"cn changes": {
			f: func(cert certificate) certificate {
				cert.cn = "new-test-cn"
				return cert
			},
			expectedResult: true,
		},
		"dnsNames changes": {
			f: func(cert certificate) certificate {
				cert.dnsNames = []string{
					"hostname1",
					"hostname2",
					"hostname3",
				}
				return cert
			},
			expectedResult: true,
		},
		"organization changes": {
			f: func(cert certificate) certificate {
				cert.organization = []string{"new-corp"}
				return cert
			},
			expectedResult: true,
		},
		"ExtKeyUsage changes": {
			f: func(cert certificate) certificate {
				cert.extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
				return cert
			},
			expectedResult: true,
		},
		"ipAddress changes": {
			f: func(cert certificate) certificate {
				cert.ipAddresses = []net.IP{net.ParseIP("192.168.0.2")}
				return cert
			},
			expectedResult: true,
		},
		"signinKey changes": {
			f: func(cert certificate) certificate {
				signingKey, signingCert, err := tls.NewCA("new-test-ca")
				if err != nil {
					t.Fatal(err)
				}
				cert.signingCert = signingCert
				cert.signingKey = signingKey
				return cert
			},
			expectedResult: true,
		},
	}

	for name, test := range tests {
		var c certificate
		if test.f != nil {
			c = test.f(cert)
		}
		if needsGenerate(c) != test.expectedResult {
			t.Fatalf("test %s failed", name)
		}
	}
}
