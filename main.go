package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"os"
	"reflect"
	"unicode"

	"github.com/jim-minter/azure-helm/pkg/tls"
	"gopkg.in/yaml.v2"
)

type config struct {
	// etcd certificates
	EtcdCaKey      *rsa.PrivateKey
	EtcdCaCert     *x509.Certificate
	EtcdServerKey  *rsa.PrivateKey
	EtcdServerCert *x509.Certificate
	EtcdPeerKey    *rsa.PrivateKey
	EtcdPeerCert   *x509.Certificate

	// control plane certificates
	MasterServerCaKey        *rsa.PrivateKey
	MasterServerCaCert       *x509.Certificate
	MasterServerKey          *rsa.PrivateKey
	MasterServerCert         *x509.Certificate
	Front_ProxyCaKey         *rsa.PrivateKey
	Front_ProxyCaCert        *x509.Certificate
	FrontProxyCaKey          *rsa.PrivateKey
	FrontProxyCaCert         *x509.Certificate
	ServiceServingKey        *rsa.PrivateKey
	ServiceServingCert       *x509.Certificate
	AdminKey                 *rsa.PrivateKey
	AdminCert                *x509.Certificate
	AggregatorFrontProxyKey  *rsa.PrivateKey
	AggregatorFrontProxyCert *x509.Certificate
	MasterKubeletClientKey   *rsa.PrivateKey
	MasterKubeletClientCert  *x509.Certificate
	MasterProxyClientKey     *rsa.PrivateKey
	MasterProxyClientCert    *x509.Certificate

	// azure config
	TenantID        string
	SubscriptionID  string
	AadClientID     string
	AadClientSecret string
	AadTenantID     string
	ResourceGroup   string
}

func (c config) MarshalYAML() (interface{}, error) {
	m := map[string]interface{}{}
	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		k := v.Type().Field(i).Name
		k = string(unicode.ToLower(rune(k[0]))) + k[1:]

		switch v := v.Field(i).Interface().(type) {
		case (*x509.Certificate):
			b, err := tls.CertAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)
		case (*rsa.PrivateKey):
			b, err := tls.PrivateKeyAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)
		default:
			m[k] = v
		}
	}
	return m, nil
}

var c config

func run() (err error) {
	// Generate etcd certs
	if c.EtcdCaKey, c.EtcdCaCert, err = tls.NewCA("etcd-signer"); err != nil {
		return
	}
	if c.EtcdServerKey, c.EtcdServerCert, err = tls.NewCert("master-etcd", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}
	if c.EtcdPeerKey, c.EtcdPeerCert, err = tls.NewCert("etcd-peer", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}

	// Generate openshift master certs
	if c.MasterServerCaKey, c.MasterServerCaCert, err = tls.NewCA("openshift-signer"); err != nil {
		return
	}
	if c.MasterServerKey, c.MasterServerCert, err = tls.NewCert("master-server", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.MasterServerCaKey, c.MasterServerCaCert); err != nil {
		return
	}
	if c.Front_ProxyCaKey, c.Front_ProxyCaCert, err = tls.NewCA("openshift-signer"); err != nil {
		return
	}
	if c.FrontProxyCaKey, c.FrontProxyCaCert, err = tls.NewCA("aggregator-proxy-car"); err != nil {
		return
	}
	if c.ServiceServingKey, c.ServiceServingCert, err = tls.NewCA("openshift-service-serving-signer"); err != nil {
		return
	}
	adminOrg := []string{"system:cluster-admins", "system:masters"}
	if c.AdminKey, c.AdminCert, err = tls.NewCert("system:admin", adminOrg, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.MasterServerCaKey, c.MasterServerCaCert); err != nil {
		return
	}
	if c.AggregatorFrontProxyKey, c.AggregatorFrontProxyCert, err = tls.NewCert("aggregator-front-proxy", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.Front_ProxyCaKey, c.Front_ProxyCaCert); err != nil {
		return
	}
	nodeAdminOrg := []string{"system:node-admins"}
	if c.MasterKubeletClientKey, c.MasterKubeletClientCert, err = tls.NewCert("system:openshift-node-admin", nodeAdminOrg, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.MasterServerCaKey, c.MasterServerCaCert); err != nil {
		return
	}
	if c.MasterProxyClientKey, c.MasterProxyClientCert, err = tls.NewCert("system:master-proxy", nil, nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, c.MasterServerCaKey, c.MasterServerCaCert); err != nil {
		return
	}

	// azure conf
	c.TenantID = os.Getenv("TENANT_ID")
	c.SubscriptionID = os.Getenv("SUBSCRIPTION_ID")
	c.AadClientID = os.Getenv("CLIENT_ID")
	c.AadClientSecret = os.Getenv("CLIENT_SECRET")
	c.AadTenantID = os.Getenv("TENANT_ID")
	// TODO: How do we properly discover the correct resource group?
	c.ResourceGroup = os.Getenv("RESOURCE_GROUP")

	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	os.Stdout.Write(b)

	return
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
