package main

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"io/ioutil"
	"os"

	"github.com/openshift/openshift-azure/pkg/util/tls"
)

func write(prefix string, key *rsa.PrivateKey, cert *x509.Certificate) error {
	b, err := tls.PrivateKeyAsBytes(key)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(prefix+".key", b, 0400)
	if err != nil {
		return err
	}

	b, err = tls.CertAsBytes(cert)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(prefix+".pem", b, 0666)
}

func run() error {
	/* #nosec - hasn't heard about the umask  */
	err := os.MkdirAll("secrets", 0777)
	if err != nil {
		return err
	}

	cakey, cacert, err := tls.NewCA("proxy-ca")
	if err != nil {
		return err
	}

	err = write("secrets/proxy-ca", cakey, cacert)
	if err != nil {
		return err
	}

	serverkey, servercert, err := tls.NewCert(&tls.CertParams{
		Subject: pkix.Name{CommonName: "proxy-server"},
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		SigningKey:  cakey,
		SigningCert: cacert,
	})
	if err != nil {
		return err
	}

	err = write("secrets/proxy-server", serverkey, servercert)
	if err != nil {
		return err
	}

	clientkey, clientcert, err := tls.NewCert(&tls.CertParams{
		Subject: pkix.Name{CommonName: "proxy-client"},
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
		SigningKey:  cakey,
		SigningCert: cacert,
	})
	if err != nil {
		return err
	}

	return write("secrets/proxy-client", clientkey, clientcert)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
