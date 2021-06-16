package config

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net"

	uuid "github.com/satori/go.uuid"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	api "github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/util/enrich"
	"github.com/openshift/openshift-azure/pkg/util/kubeconfig"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

func (g *simpleGenerator) Generate(template *pluginapi.Config, setVersionFields bool) (err error) {
	config, err := pluginapi.ToInternal(template, &g.cs.Config, setVersionFields)
	if err != nil {
		return err
	}

	if g.cs.Properties.PrivateAPIServer && (g.cs.Properties.FQDN == "" || g.cs.Properties.PublicHostname == "") {
		err := enrich.PrivateAPIServerIPAddress(g.cs)
		if err != nil {
			return err
		}
	}

	g.cs.Config = *config
	c := &g.cs.Config

	// Generate CAs
	cas := []struct {
		cn   string
		key  **rsa.PrivateKey
		cert **x509.Certificate
	}{
		{
			cn:   "etcd-signer",
			key:  &c.Certificates.EtcdCa.Key,
			cert: &c.Certificates.EtcdCa.Cert,
		},
		{
			cn:   "openshift-signer",
			key:  &c.Certificates.Ca.Key,
			cert: &c.Certificates.Ca.Cert,
		},
		{
			cn:   "openshift-frontproxy-signer",
			key:  &c.Certificates.FrontProxyCa.Key,
			cert: &c.Certificates.FrontProxyCa.Cert,
		},
		{
			cn:   "openshift-service-serving-signer",
			key:  &c.Certificates.ServiceSigningCa.Key,
			cert: &c.Certificates.ServiceSigningCa.Cert,
		},
		{
			cn:   "service-catalog-signer",
			key:  &c.Certificates.ServiceCatalogCa.Key,
			cert: &c.Certificates.ServiceCatalogCa.Cert,
		},
	}
	for _, ca := range cas {
		if *ca.key != nil && *ca.cert != nil {
			continue
		}
		if *ca.key, *ca.cert, err = tls.NewCA(ca.cn); err != nil {
			return
		}
	}

	masterParams := tls.CertParams{
		Subject: pkix.Name{
			CommonName: g.cs.Properties.FQDN,
		},
		DNSNames: []string{
			"master-000000",
			"master-000001",
			"master-000002",
			"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			"kubernetes.default.svc.cluster.local",
		},
		IPAddresses: []net.IP{
			net.ParseIP("172.30.0.1"),
		},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if g.cs.Properties.PrivateAPIServer {
		masterParams.IPAddresses = append(masterParams.IPAddresses, net.ParseIP(g.cs.Properties.FQDN))
	} else {
		masterParams.DNSNames = append(masterParams.DNSNames, g.cs.Properties.FQDN)
	}

	certs := []struct {
		params tls.CertParams
		key    **rsa.PrivateKey
		cert   **x509.Certificate
	}{
		// Generate etcd certs
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "etcd-server",
				},
				DNSNames:    []string{"master-000000", "master-000001", "master-000002"},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				SigningKey:  c.Certificates.EtcdCa.Key,
				SigningCert: c.Certificates.EtcdCa.Cert,
			},
			key:  &c.Certificates.EtcdServer.Key,
			cert: &c.Certificates.EtcdServer.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "etcd-peer",
				},
				DNSNames:    []string{"master-000000", "master-000001", "master-000002"},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
				SigningKey:  c.Certificates.EtcdCa.Key,
				SigningCert: c.Certificates.EtcdCa.Cert,
			},
			key:  &c.Certificates.EtcdPeer.Key,
			cert: &c.Certificates.EtcdPeer.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "etcd-client",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				SigningKey:  c.Certificates.EtcdCa.Key,
				SigningCert: c.Certificates.EtcdCa.Cert,
			},
			key:  &c.Certificates.EtcdClient.Key,
			cert: &c.Certificates.EtcdClient.Cert,
		},
		// Generate openshift master certs
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName:   "system:admin",
					Organization: []string{"system:cluster-admins", "system:masters"},
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.Admin.Key,
			cert: &c.Certificates.Admin.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "aggregator-front-proxy",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				SigningKey:  c.Certificates.FrontProxyCa.Key,
				SigningCert: c.Certificates.FrontProxyCa.Cert,
			},
			key:  &c.Certificates.AggregatorFrontProxy.Key,
			cert: &c.Certificates.AggregatorFrontProxy.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{CommonName: "system:openshift-node-admin",
					Organization: []string{"system:node-admins"},
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.MasterKubeletClient.Key,
			cert: &c.Certificates.MasterKubeletClient.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "system:master-proxy",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.MasterProxyClient.Key,
			cert: &c.Certificates.MasterProxyClient.Cert,
		},
		{
			params: masterParams,
			key:    &c.Certificates.MasterServer.Key,
			cert:   &c.Certificates.MasterServer.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{CommonName: "system:openshift-master",
					Organization: []string{"system:masters"},
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.OpenShiftMaster.Key,
			cert: &c.Certificates.OpenShiftMaster.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "apiserver.kube-service-catalog.svc",
				},
				DNSNames: []string{
					"apiserver.kube-service-catalog.svc",
					"apiserver.kube-service-catalog.svc.cluster.local",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				SigningKey:  c.Certificates.ServiceCatalogCa.Key,
				SigningCert: c.Certificates.ServiceCatalogCa.Cert,
			},
			key:  &c.Certificates.ServiceCatalogServer.Key,
			cert: &c.Certificates.ServiceCatalogServer.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "aro-admission-controller.kube-system.svc",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
			key:  &c.Certificates.AroAdmissionController.Key,
			cert: &c.Certificates.AroAdmissionController.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "aro-admission-controller-client",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.AroAdmissionControllerClient.Key,
			cert: &c.Certificates.AroAdmissionControllerClient.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "system:serviceaccount:openshift-infra:node-bootstrapper",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.NodeBootstrap.Key,
			cert: &c.Certificates.NodeBootstrap.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "system:serviceaccount:openshift-sdn:sdn",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.SDN.Key,
			cert: &c.Certificates.SDN.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "system:serviceaccount:openshift-azure:blackboxmonitor",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			key:  &c.Certificates.BlackBoxMonitor.Key,
			cert: &c.Certificates.BlackBoxMonitor.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "docker-registry.default.svc",
				},
				DNSNames: []string{
					"docker-registry.default.svc",
					"docker-registry.default.svc.cluster.local",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
			key:  &c.Certificates.Registry.Key,
			cert: &c.Certificates.Registry.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "registry-console.default.svc",
				},
				DNSNames: []string{
					"registry-console.default.svc",
					"registry-console.default.svc.cluster.local",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
			key:  &c.Certificates.RegistryConsole.Key,
			cert: &c.Certificates.RegistryConsole.Cert,
		},
		{
			params: tls.CertParams{
				Subject: pkix.Name{
					CommonName: "metrics-server.openshift-monitoring.svc",
				},
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
			key:  &c.Certificates.MetricsServer.Key,
			cert: &c.Certificates.MetricsServer.Cert,
		},
	}
	for _, cert := range certs {
		if cert.params.SigningKey == nil && cert.params.SigningCert == nil {
			cert.params.SigningKey, cert.params.SigningCert = c.Certificates.Ca.Key, c.Certificates.Ca.Cert
		}
		if !tls.CertMatchesParams(*cert.key, *cert.cert, &cert.params) {
			if *cert.key, *cert.cert, err = tls.NewCert(&cert.params); err != nil {
				return
			}
		}
	}

	secrets := []struct {
		secret *[]byte
		n      int
	}{
		{
			secret: &c.SessionSecretAuth,
			n:      24,
		},
		{
			secret: &c.SessionSecretEnc,
			n:      24,
		},
		{
			secret: &c.RegistryHTTPSecret,
		},
		{
			secret: &c.AlertManagerProxySessionSecret,
		},
		{
			secret: &c.AlertsProxySessionSecret,
		},
		{
			secret: &c.PrometheusProxySessionSecret,
		},
	}
	for _, s := range secrets {
		if len(*s.secret) != 0 {
			continue
		}
		if s.n == 0 {
			s.n = 32
		}
		if *s.secret, err = random.Bytes(s.n); err != nil {
			return
		}
	}

	kubeconfigs := []struct {
		clientKey  *rsa.PrivateKey
		clientCert *x509.Certificate
		endpoint   string
		username   string
		namespace  string
		kubeconfig **v1.Config
	}{
		{
			clientKey:  c.Certificates.OpenShiftMaster.Key,
			clientCert: c.Certificates.OpenShiftMaster.Cert,
			endpoint:   g.cs.Properties.FQDN,
			username:   "system:openshift-master",
			kubeconfig: &c.MasterKubeconfig,
		},
		{
			clientKey:  c.Certificates.Admin.Key,
			clientCert: c.Certificates.Admin.Cert,
			endpoint:   g.cs.Properties.FQDN,
			username:   "system:admin",
			kubeconfig: &c.AdminKubeconfig,
		},
		{
			clientKey:  c.Certificates.NodeBootstrap.Key,
			clientCert: c.Certificates.NodeBootstrap.Cert,
			endpoint:   g.cs.Properties.FQDN,
			username:   "system:serviceaccount:openshift-infra:node-bootstrapper",
			kubeconfig: &c.NodeBootstrapKubeconfig,
			namespace:  "openshift-infra",
		},
		{
			clientKey:  c.Certificates.SDN.Key,
			clientCert: c.Certificates.SDN.Cert,
			endpoint:   g.cs.Properties.FQDN,
			username:   "system:serviceaccount:openshift-sdn:sdn",
			kubeconfig: &c.SDNKubeconfig,
			namespace:  "openshift-sdn",
		},
		{
			clientKey:  c.Certificates.BlackBoxMonitor.Key,
			clientCert: c.Certificates.BlackBoxMonitor.Cert,
			endpoint:   g.cs.Properties.FQDN,
			username:   "system:serviceaccount:openshift-azure:blackboxmonitor",
			kubeconfig: &c.BlackBoxMonitorKubeconfig,
			namespace:  "openshift-azure",
		},
	}
	for _, kc := range kubeconfigs {
		if kc.namespace == "" {
			kc.namespace = "default"
		}
		if *kc.kubeconfig, err = kubeconfig.Make(kc.clientKey, kc.clientCert, c.Certificates.Ca.Cert, kc.endpoint, kc.username, kc.namespace); err != nil {
			return
		}
	}

	if c.ServiceAccountKey == nil {
		if c.ServiceAccountKey, err = tls.NewPrivateKey(); err != nil {
			return
		}
	}

	if c.SSHKey == nil {
		if c.SSHKey, err = tls.NewPrivateKey(); err != nil {
			return
		}
	}

	if len(c.RegistryStorageAccount) == 0 {
		if c.RegistryStorageAccount, err = random.StorageAccountName("sareg"); err != nil {
			return
		}
	}

	if len(c.AzureFileStorageAccount) == 0 {
		if c.AzureFileStorageAccount, err = random.StorageAccountName("safil"); err != nil {
			return
		}
	}

	if len(c.ConfigStorageAccount) == 0 {
		if c.ConfigStorageAccount, err = random.StorageAccountName("sacfg"); err != nil {
			return
		}
	}

	if len(c.RegistryConsoleOAuthSecret) == 0 {
		var pass string
		if pass, err = random.AlphanumericString(64); err != nil {
			return err
		}
		c.RegistryConsoleOAuthSecret = fmt.Sprintf("user%s", pass)
	}

	if len(c.ConsoleOAuthSecret) == 0 {
		if c.ConsoleOAuthSecret, err = random.AlphanumericString(64); err != nil {
			return err
		}
	}

	if len(c.RouterStatsPassword) == 0 {
		if c.RouterStatsPassword, err = random.AlphanumericString(10); err != nil {
			return
		}
	}

	if len(c.EtcdMetricsPassword) == 0 {
		if c.EtcdMetricsPassword, err = random.AlphanumericString(10); err != nil {
			return
		}
	}
	if len(c.EtcdMetricsUsername) == 0 {
		if c.EtcdMetricsUsername, err = random.AlphanumericString(10); err != nil {
			return
		}
	}

	if uuid.Equal(c.ServiceCatalogClusterID, uuid.Nil) {
		c.ServiceCatalogClusterID = uuid.NewV4()
	}

	return
}

// InvalidateCertificates removes some certificates from an OpenShiftManagedCluster.Config
func (g *simpleGenerator) InvalidateCertificates() (err error) {
	g.cs.Config.Certificates.EtcdClient = api.CertKeyPair{}
	g.cs.Config.Certificates.EtcdServer = api.CertKeyPair{}
	g.cs.Config.Certificates.EtcdPeer = api.CertKeyPair{}
	g.cs.Config.Certificates.Admin = api.CertKeyPair{}
	g.cs.Config.Certificates.OpenShiftMaster = api.CertKeyPair{}
	g.cs.Config.Certificates.ServiceCatalogCa = api.CertKeyPair{}
	g.cs.Config.Certificates.AroAdmissionController = api.CertKeyPair{}
	g.cs.Config.Certificates.AroAdmissionControllerClient = api.CertKeyPair{}
	g.cs.Config.Certificates.NodeBootstrap = api.CertKeyPair{}
	g.cs.Config.Certificates.BlackBoxMonitor = api.CertKeyPair{}
	g.cs.Config.Certificates.RegistryConsole = api.CertKeyPair{}
	g.cs.Config.Certificates.Registry = api.CertKeyPair{}
	g.cs.Config.Certificates.FrontProxyCa = api.CertKeyPair{}
	g.cs.Config.Certificates.MasterKubeletClient = api.CertKeyPair{}
	g.cs.Config.Certificates.MetricsServer = api.CertKeyPair{}
	g.cs.Config.Certificates.MasterProxyClient = api.CertKeyPair{}
	g.cs.Config.Certificates.MasterServer = api.CertKeyPair{}
	g.cs.Config.Certificates.AggregatorFrontProxy = api.CertKeyPair{}

	return nil
}

// InvalidateSecrets removes some secrets from an OpenShiftManagedCluster.Config
func (g *simpleGenerator) InvalidateSecrets() (err error) {
	g.cs.Config.SSHKey = nil

	g.cs.Config.Certificates.GenevaLogging = api.CertKeyPair{}
	g.cs.Config.Certificates.GenevaMetrics = api.CertKeyPair{}
	g.cs.Config.Certificates.PackageRepository = api.CertKeyPair{}

	g.cs.Config.Images.GenevaImagePullSecret = nil
	g.cs.Config.Images.ImagePullSecret = nil

	g.cs.Config.SessionSecretAuth = nil
	g.cs.Config.SessionSecretEnc = nil

	g.cs.Config.RegistryHTTPSecret = nil
	g.cs.Config.PrometheusProxySessionSecret = nil
	g.cs.Config.AlertManagerProxySessionSecret = nil
	g.cs.Config.AlertsProxySessionSecret = nil
	g.cs.Config.RegistryConsoleOAuthSecret = ""
	g.cs.Config.ConsoleOAuthSecret = ""
	g.cs.Config.RouterStatsPassword = ""
	g.cs.Config.EtcdMetricsPassword = ""
	g.cs.Config.EtcdMetricsUsername = ""

	return
}

func (g *simpleGenerator) GenerateStartup() (cs *api.OpenShiftManagedCluster, err error) {
	return g.cs, nil
}
