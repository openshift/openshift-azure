package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"strings"
	"unicode"

	"github.com/ghodss/yaml"
	"github.com/jim-minter/azure-helm/pkg/tls"
	"github.com/satori/uuid"
	"golang.org/x/crypto/bcrypt"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

type config struct {
	// CAs
	EtcdCaKey            *rsa.PrivateKey
	EtcdCaCert           *x509.Certificate
	CaKey                *rsa.PrivateKey
	CaCert               *x509.Certificate
	FrontProxyCaKey      *rsa.PrivateKey
	FrontProxyCaCert     *x509.Certificate
	ServiceSigningCaKey  *rsa.PrivateKey
	ServiceSigningCaCert *x509.Certificate
	ServiceCatalogCaKey  *rsa.PrivateKey
	ServiceCatalogCaCert *x509.Certificate

	// container images for pods
	MasterEtcdImage            string
	MasterAPIImage             string
	MasterControllersImage     string
	BootstrapAutoapproverImage string
	ServiceCatalogImage        string
	ImportImage                string

	// etcd certificates
	EtcdServerKey  *rsa.PrivateKey
	EtcdServerCert *x509.Certificate
	EtcdPeerKey    *rsa.PrivateKey
	EtcdPeerCert   *x509.Certificate
	EtcdClientKey  *rsa.PrivateKey
	EtcdClientCert *x509.Certificate

	// control plane certificates
	MasterServerKey          *rsa.PrivateKey
	MasterServerCert         *x509.Certificate
	AdminKey                 *rsa.PrivateKey
	AdminCert                *x509.Certificate
	AggregatorFrontProxyKey  *rsa.PrivateKey
	AggregatorFrontProxyCert *x509.Certificate
	MasterKubeletClientKey   *rsa.PrivateKey
	MasterKubeletClientCert  *x509.Certificate
	MasterProxyClientKey     *rsa.PrivateKey
	MasterProxyClientCert    *x509.Certificate
	OpenShiftMasterKey       *rsa.PrivateKey
	OpenShiftMasterCert      *x509.Certificate

	ServiceCatalogServerKey  *rsa.PrivateKey
	ServiceCatalogServerCert *x509.Certificate

	// master-config configurables
	RoutingConfigSubdomain string
	PublicHostname         string
	ImageConfigFormat      string

	// misc control plane configurables
	ServiceAccountPrivateKey *rsa.PrivateKey
	ServiceAccountPublicKey  *rsa.PublicKey
	SessionSecretAuth        []byte
	SessionSecretEnc         []byte
	HtPasswd                 []byte

	// kubeconfigs
	AdminKubeconfig  *v1.Config
	MasterKubeconfig *v1.Config

	// azure config
	TenantID        string
	SubscriptionID  string
	AadClientID     string
	AadClientSecret string
	AadTenantID     string
	ResourceGroup   string
	Location        string

	// needed by import
	RouterIP                       net.IP
	EtcdHostname                   string
	RegistryStorageAccount         string
	RegistryAccountKey             string
	RegistryServiceIP              net.IP
	RegistryHTTPSecret             []byte
	AlertManagerProxySessionSecret []byte
	AlertsProxySessionSecret       []byte
	PrometheusProxySessionSecret   []byte
	ServiceCatalogClusterID        uuid.UUID
	RegistryKey                    *rsa.PrivateKey
	RegistryCert                   *x509.Certificate
	RouterKey                      *rsa.PrivateKey
	RouterCert                     *x509.Certificate
}

func (c config) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		k := v.Type().Field(i).Name
		k = string(unicode.ToLower(rune(k[0]))) + k[1:]

		switch v := v.Field(i).Interface().(type) {
		case *x509.Certificate:
			b, err := tls.CertAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case *rsa.PrivateKey:
			b, err := tls.PrivateKeyAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case *rsa.PublicKey:
			b, err := tls.PublicKeyAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case *v1.Config:
			b, err := yaml.Marshal(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case []byte:
			m[k] = base64.StdEncoding.EncodeToString(v)

		default:
			m[k] = v
		}
	}
	return json.Marshal(m)
}

var c config

func run() (err error) {
	// Eventually these will be passed in from OSA config
	c.Location = "eastus"
	c.ResourceGroup = "jminterhcp"

	c.RoutingConfigSubdomain = "example.com"
	c.PublicHostname = "master-api-demo.104.45.157.35.nip.io"
	c.ImageConfigFormat = "openshift/origin-${component}:${version}"

	c.MasterEtcdImage = "quay.io/coreos/etcd:v3.2.15"
	c.MasterAPIImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.MasterControllersImage = "docker.io/openshift/origin-control-plane:v3.10"
	c.BootstrapAutoapproverImage = "docker.io/openshift/origin-node:v3.10.0"
	c.ServiceCatalogImage = "docker.io/openshift/origin-service-catalog:v3.10.0"
	c.ImportImage = "docker.io/jimminter/import:latest"

	// TODO: need to cross-check all the below with acs-engine, especially SANs and IPs

	// Generate CAs
	if c.EtcdCaKey, c.EtcdCaCert, err = tls.NewCA("etcd-signer"); err != nil {
		return
	}
	if c.CaKey, c.CaCert, err = tls.NewCA("openshift-signer"); err != nil {
		return
	}
	// currently skipping the other frontproxy, doesn't seem to hurt
	if c.FrontProxyCaKey, c.FrontProxyCaCert, err = tls.NewCA("openshift-frontproxy-signer"); err != nil {
		return
	}
	if c.ServiceSigningCaKey, c.ServiceSigningCaCert, err = tls.NewCA("openshift-service-serving-signer"); err != nil {
		return
	}
	if c.ServiceCatalogCaKey, c.ServiceCatalogCaCert, err = tls.NewCA("service-catalog-signer"); err != nil {
		return
	}

	// Generate etcd certs
	if c.EtcdServerKey, c.EtcdServerCert, err = tls.NewCert("master-etcd", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}
	if c.EtcdPeerKey, c.EtcdPeerCert, err = tls.NewCert("etcd-peer", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}
	if c.EtcdClientKey, c.EtcdClientCert, err = tls.NewCert("etcd-client", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}

	// Generate openshift master certs
	if c.AdminKey, c.AdminCert, err = tls.NewCert("system:admin", []string{"system:cluster-admins", "system:masters"}, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	if c.AggregatorFrontProxyKey, c.AggregatorFrontProxyCert, err = tls.NewCert("aggregator-front-proxy", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.FrontProxyCaKey, c.FrontProxyCaCert); err != nil {
		return
	}
	// currently skipping etcd.server, doesn't seem to hurt
	if c.MasterKubeletClientKey, c.MasterKubeletClientCert, err = tls.NewCert("system:openshift-node-admin", []string{"system:node-admins"}, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	if c.MasterProxyClientKey, c.MasterProxyClientCert, err = tls.NewCert("system:master-proxy", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	if c.MasterServerKey, c.MasterServerCert, err = tls.NewCert("master-api", nil, []string{"master-api", c.PublicHostname}, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}
	// currently skipping openshift-aggregator, doesn't seem to hurt
	if c.OpenShiftMasterKey, c.OpenShiftMasterCert, err = tls.NewCert("system:openshift-master", []string{"system:cluster-admins", "system:masters"}, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.CaKey, c.CaCert); err != nil {
		return
	}

	if c.ServiceCatalogServerKey, c.ServiceCatalogServerCert, err = tls.NewCert("servicecatalog-api", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.ServiceCatalogCaKey, c.ServiceCatalogCaCert); err != nil {
		return
	}

	if c.ServiceAccountPrivateKey, err = tls.NewPrivateKey(); err != nil {
		return
	}
	c.ServiceAccountPublicKey = &c.ServiceAccountPrivateKey.PublicKey

	if c.SessionSecretAuth, err = randomBytes(24); err != nil {
		return
	}
	if c.SessionSecretEnc, err = randomBytes(24); err != nil {
		return
	}

	if c.HtPasswd, err = makeHtPasswd("demo", "demo"); err != nil {
		return
	}

	// azure conf
	c.TenantID = os.Getenv("AZURE_TENANT_ID")
	c.SubscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	c.AadClientID = os.Getenv("AZURE_CLIENT_ID")
	c.AadClientSecret = os.Getenv("AZURE_CLIENT_SECRET")
	c.AadTenantID = os.Getenv("AZURE_TENANT_ID")

	c.MasterKubeconfig = makeKubeConfig(c.OpenShiftMasterKey, c.OpenShiftMasterCert, c.CaCert, "master-api", "system:openshift-master", "default")
	c.AdminKubeconfig = makeKubeConfig(c.AdminKey, c.AdminCert, c.CaCert, c.PublicHostname, "system:admin", "default")

	// needed by import
	// TODO: these need to be filled out sanely, and need to fully migrate the
	// service catalog over from impexp to helm.
	c.RouterIP = net.ParseIP("0.0.0.0")
	c.EtcdHostname = "garbage"
	c.RegistryStorageAccount = "garbage"
	c.RegistryAccountKey = "garbage"
	c.RegistryServiceIP = net.ParseIP("172.30.190.177") // TODO: choose a particular IP address?
	if c.RegistryHTTPSecret, err = randomBytes(32); err != nil {
		return err
	}
	if c.AlertManagerProxySessionSecret, err = randomBytes(32); err != nil {
		return err
	}
	if c.AlertsProxySessionSecret, err = randomBytes(32); err != nil {
		return err
	}
	if c.PrometheusProxySessionSecret, err = randomBytes(32); err != nil {
		return err
	}
	if c.ServiceCatalogClusterID, err = uuid.NewV4(); err != nil {
		return err
	}
	// TODO: is it possible for the registry to use
	// service.alpha.openshift.io/serving-cert-secret-name?
	// TODO: remove nip.io
	c.RegistryKey, c.RegistryCert, err =
		tls.NewCert(c.RegistryServiceIP.String(), nil,
			[]string{"docker-registry-default." + c.RegistryServiceIP.String() + ".nip.io",
				"docker-registry.default.svc",
				"docker-registry.default.svc.cluster.local",
			},
			[]net.IP{c.RegistryServiceIP},
			[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			c.CaKey,
			c.CaCert)
	if err != nil {
		return err
	}
	// TODO: the router CN and SANs should be configurables.
	c.RouterKey, c.RouterCert, err =
		tls.NewCert("*."+c.RouterIP.String()+".nip.io", nil,
			[]string{"*." + c.RouterIP.String() + ".nip.io",
				c.RouterIP.String() + ".nip.io",
			},
			nil,
			[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			c.CaKey,
			c.CaCert)
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	os.Stdout.Write(b)

	b, err = yaml.Marshal(c.AdminKubeconfig)
	if err != nil {
		return
	}

	return ioutil.WriteFile("admin.kubeconfig", b, 0600)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func makeKubeConfig(clientKey *rsa.PrivateKey, clientCert, caCert *x509.Certificate, endpoint, username, namespace string) *v1.Config {
	clustername := strings.Replace(endpoint, ".", "-", -1)
	authinfoname := username + "/" + clustername
	contextname := namespace + "/" + clustername + "/" + username

	return &v1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []v1.NamedCluster{
			{
				Name: clustername,
				Cluster: v1.Cluster{
					Server: "https://" + endpoint,
					CertificateAuthorityData: tls.MustCertAsBytes(caCert),
				},
			},
		},
		AuthInfos: []v1.NamedAuthInfo{
			{
				Name: authinfoname,
				AuthInfo: v1.AuthInfo{
					ClientCertificateData: tls.MustCertAsBytes(clientCert),
					ClientKeyData:         tls.MustPrivateKeyAsBytes(clientKey),
				},
			},
		},
		Contexts: []v1.NamedContext{
			{
				Name: contextname,
				Context: v1.Context{
					Cluster:   clustername,
					Namespace: namespace,
					AuthInfo:  authinfoname,
				},
			},
		},
		CurrentContext: contextname,
	}
}

func makeHtPasswd(username, password string) ([]byte, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return append([]byte(username+":"), b...), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadAtLeast(rand.Reader, b, n); err != nil {
		return nil, err
	}
	return b, nil
}
