package config

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"reflect"

	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	api "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/tls"
)

type certificate struct {
	cn           string
	organization []string
	dnsNames     []string
	ipAddresses  []net.IP
	extKeyUsage  []x509.ExtKeyUsage
	signingKey   *rsa.PrivateKey
	signingCert  *x509.Certificate
	key          **rsa.PrivateKey
	cert         **x509.Certificate
}

func Generate(cs *api.OpenShiftManagedCluster, pluginConfig api.PluginConfig) (err error) {
	c := cs.Config

	selectNodeImage(cs, os.Getenv("DEPLOY_OS"))

	if err = selectContainerImages(cs, pluginConfig); err != nil {
		return err
	}

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

	certs := []certificate{
		{
			// Generate etcd certs
			cn:          "etcd-server",
			dnsNames:    []string{"master-000000", "master-000001", "master-000002"},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			signingKey:  c.Certificates.EtcdCa.Key,
			signingCert: c.Certificates.EtcdCa.Cert,
			key:         &c.Certificates.EtcdServer.Key,
			cert:        &c.Certificates.EtcdServer.Cert,
		},
		{
			cn:          "etcd-peer",
			dnsNames:    []string{"master-000000", "master-000001", "master-000002"},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			signingKey:  c.Certificates.EtcdCa.Key,
			signingCert: c.Certificates.EtcdCa.Cert,
			key:         &c.Certificates.EtcdPeer.Key,
			cert:        &c.Certificates.EtcdPeer.Cert,
		},
		{
			cn:          "etcd-client",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			signingKey:  c.Certificates.EtcdCa.Key,
			signingCert: c.Certificates.EtcdCa.Cert,
			key:         &c.Certificates.EtcdClient.Key,
			cert:        &c.Certificates.EtcdClient.Cert,
		},
		// Generate openshift master certs
		{
			cn:           "system:admin",
			organization: []string{"system:cluster-admins", "system:masters"},
			extKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:          &c.Certificates.Admin.Key,
			cert:         &c.Certificates.Admin.Cert,
		},
		{
			cn:          "aggregator-front-proxy",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			signingKey:  c.Certificates.FrontProxyCa.Key,
			signingCert: c.Certificates.FrontProxyCa.Cert,
			key:         &c.Certificates.AggregatorFrontProxy.Key,
			cert:        &c.Certificates.AggregatorFrontProxy.Cert,
		},
		{
			cn:           "system:openshift-node-admin",
			organization: []string{"system:node-admins"},
			extKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:          &c.Certificates.MasterKubeletClient.Key,
			cert:         &c.Certificates.MasterKubeletClient.Cert,
		},
		{
			cn:          "system:master-proxy",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.Certificates.MasterProxyClient.Key,
			cert:        &c.Certificates.MasterProxyClient.Cert,
		},
		{
			cn: cs.Properties.FQDN,
			dnsNames: []string{
				cs.Properties.FQDN,
				"master-000000",
				"master-000001",
				"master-000002",
				"kubernetes",
				"kubernetes.default",
				"kubernetes.default.svc",
				"kubernetes.default.svc.cluster.local",
			},
			ipAddresses: []net.IP{net.ParseIP("172.30.0.1")},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.Certificates.MasterServer.Key,
			cert:        &c.Certificates.MasterServer.Cert,
		},
		{
			cn:           "system:openshift-master",
			organization: []string{"system:cluster-admins", "system:masters"},
			extKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:          &c.Certificates.OpenShiftMaster.Key,
			cert:         &c.Certificates.OpenShiftMaster.Cert,
		},
		{
			cn: "servicecatalog-api",
			dnsNames: []string{
				"servicecatalog-api",
				"apiserver.kube-service-catalog.svc", // TODO: unclear how safe this is
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			signingKey:  c.Certificates.ServiceCatalogCa.Key,
			signingCert: c.Certificates.ServiceCatalogCa.Cert,
			key:         &c.Certificates.ServiceCatalogServer.Key,
			cert:        &c.Certificates.ServiceCatalogServer.Cert,
		},
		{
			cn:          "system:serviceaccount:kube-service-catalog:service-catalog-apiserver",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.Certificates.ServiceCatalogAPIClient.Key,
			cert:        &c.Certificates.ServiceCatalogAPIClient.Cert,
		},
		{
			cn:          "system:serviceaccount:openshift-infra:node-bootstrapper",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.Certificates.NodeBootstrap.Key,
			cert:        &c.Certificates.NodeBootstrap.Cert,
		},
		{
			cn:          "system:serviceaccount:openshift-azure:azure-cluster-reader",
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			key:         &c.Certificates.AzureClusterReader.Key,
			cert:        &c.Certificates.AzureClusterReader.Cert,
		},
		{
			cn: cs.Properties.RouterProfiles[0].PublicSubdomain,
			dnsNames: []string{
				cs.Properties.RouterProfiles[0].PublicSubdomain,
				"*." + cs.Properties.RouterProfiles[0].PublicSubdomain,
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.Certificates.Router.Key,
			cert:        &c.Certificates.Router.Cert,
		},
		{
			cn: "docker-registry-default." + cs.Properties.RouterProfiles[0].PublicSubdomain,
			dnsNames: []string{
				"docker-registry-default." + cs.Properties.RouterProfiles[0].PublicSubdomain,
				"docker-registry.default.svc",
				"docker-registry.default.svc.cluster.local",
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.Certificates.Registry.Key,
			cert:        &c.Certificates.Registry.Cert,
		},
		// Openshift Console is BYO type of certificate. In the long run we should
		// enable users to configure their own certificates.
		// For this reason we decouple it from all OCP certs and make it self-sign
		// If FQDN matches PublicHostname certificate can't be self-sign
		// https://github.com/openshift/openshift-azure/issues/307
		{
			cn: Derived.PublicHostname(cs),
			dnsNames: []string{
				Derived.PublicHostname(cs),
			},
			extKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			key:         &c.Certificates.OpenshiftConsole.Key,
			cert:        &c.Certificates.OpenshiftConsole.Cert,
		},
	}
	for _, cert := range certs {
		if cert.signingKey == nil && cert.signingCert == nil {
			cert.signingKey, cert.signingCert = c.Certificates.Ca.Key, c.Certificates.Ca.Cert
		}
		var k *rsa.PrivateKey
		var c *x509.Certificate
		if k, c, err = tls.NewCert(cert.cn, cert.organization, cert.dnsNames, cert.ipAddresses, cert.extKeyUsage, cert.signingKey, cert.signingCert, false); err != nil {
			return
		}
		// compare certificate and replace if update is needed
		// if at any point we start using self-sign certificates again
		// (see console certificate comment), logic inside certEqual needs
		// to be updated for this.
		if !certEqual(c, *cert.cert) {
			*cert.key, *cert.cert = k, c
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
		if *s.secret, err = randomBytes(s.n); err != nil {
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
			endpoint:   cs.Properties.FQDN,
			username:   "system:openshift-master",
			kubeconfig: &c.MasterKubeconfig,
		},
		{
			clientKey:  c.Certificates.Admin.Key,
			clientCert: c.Certificates.Admin.Cert,
			endpoint:   cs.Properties.FQDN,
			username:   "system:admin",
			kubeconfig: &c.AdminKubeconfig,
		},
		{
			clientKey:  c.Certificates.NodeBootstrap.Key,
			clientCert: c.Certificates.NodeBootstrap.Cert,
			endpoint:   cs.Properties.FQDN,
			username:   "system:serviceaccount:openshift-infra:node-bootstrapper",
			kubeconfig: &c.NodeBootstrapKubeconfig,
			namespace:  "openshift-infra",
		},
		{
			clientKey:  c.Certificates.AzureClusterReader.Key,
			clientCert: c.Certificates.AzureClusterReader.Cert,
			endpoint:   cs.Properties.FQDN,
			username:   "system:serviceaccount:openshift-azure:azure-cluster-reader",
			kubeconfig: &c.AzureClusterReaderKubeconfig,
			namespace:  "openshift-azure",
		},
	}
	for _, kc := range kubeconfigs {
		if kc.namespace == "" {
			kc.namespace = "default"
		}
		if *kc.kubeconfig, err = makeKubeConfig(kc.clientKey, kc.clientCert, c.Certificates.Ca.Cert, kc.endpoint, kc.username, kc.namespace); err != nil {
			return
		}
	}

	if c.ServiceAccountKey == nil {
		if c.ServiceAccountKey, err = tls.NewPrivateKey(); err != nil {
			return
		}
	}

	// TODO: Remove these password operations before GA
	if len(c.AdminPasswd) == 0 {
		if c.AdminPasswd, err = randomString(10); err != nil {
			return err
		}
	}
	if len(c.HtPasswd) == 0 || bcrypt.CompareHashAndPassword(getHashFromHtPasswd(c.HtPasswd), []byte(c.AdminPasswd)) != nil {
		c.HtPasswd, err = makeHtPasswd("osadmin", c.AdminPasswd)
		if err != nil {
			return err
		}
	}

	if c.SSHKey == nil {
		if c.SSHKey, err = tls.NewPrivateKey(); err != nil {
			return
		}
	}

	if len(c.RegistryStorageAccount) == 0 {
		if c.RegistryStorageAccount, err = randomStorageAccountName(); err != nil {
			return
		}
	}

	if len(c.ConfigStorageAccount) == 0 {
		if c.ConfigStorageAccount, err = randomStorageAccountName(); err != nil {
			return
		}
	}

	if len(c.LoggingWorkspace) == 0 {
		if c.LoggingWorkspace, err = randomStorageAccountName(); err != nil {
			return
		}
	}

	if len(c.LoggingLocation) == 0 {
		c.LoggingLocation = api.AzureLocations[cs.Location]
	}

	if len(c.RegistryConsoleOAuthSecret) == 0 {
		var pass string
		if pass, err = randomString(64); err != nil {
			return err
		}
		c.RegistryConsoleOAuthSecret = fmt.Sprintf("user%s", pass)
	}

	if len(c.RouterStatsPassword) == 0 {
		if c.RouterStatsPassword, err = randomString(10); err != nil {
			return
		}
	}

	if uuid.Equal(c.ServiceCatalogClusterID, uuid.Nil) {
		c.ServiceCatalogClusterID = uuid.NewV4()
	}

	return
}

func certEqual(certNew, certOld *x509.Certificate) bool {
	if certOld == nil {
		return false
	}
	if !reflect.DeepEqual(certNew.Subject, certOld.Subject) ||
		!reflect.DeepEqual(certNew.DNSNames, certOld.DNSNames) ||
		!reflect.DeepEqual(certNew.ExtKeyUsage, certOld.ExtKeyUsage) ||
		!reflect.DeepEqual(certNew.IPAddresses, certOld.IPAddresses) ||
		!reflect.DeepEqual(certNew.Issuer, certOld.Issuer) {
		return false
	}
	return true
}
