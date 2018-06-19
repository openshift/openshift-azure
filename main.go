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
	if c.EtcdCaKey, c.EtcdCaCert, err = tls.NewCA("etcd-signer"); err != nil {
		return
	}

	if c.EtcdServerKey, c.EtcdServerCert, err = tls.NewCert("master-etcd", nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}

	if c.EtcdPeerKey, c.EtcdPeerCert, err = tls.NewCert("etcd-peer", nil, nil, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, c.EtcdCaKey, c.EtcdCaCert); err != nil {
		return
	}

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
