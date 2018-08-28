package tls

import (
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"testing"
	"time"
)

const (
	updateSSLVar = "UPDATE_KNOWN_SSL_CERT"
)

func readCert(path string) (*x509.Certificate, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseCert(b)
}

func writeCert(path string, cert *x509.Certificate) error {
	b, err := CertAsBytes(cert)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, b, 0666)
}

func TestNewPrivateKey(t *testing.T) {
	key, err := NewPrivateKey()
	if err != nil {
		t.Error(err)
	}
	if key.Validate() != nil {
		t.Error(err)
	}
	if key.N.BitLen() < 2048 {
		t.Errorf("insecure key length detected: %d", key.N.BitLen())
	}
}

func TestNewCA(t *testing.T) {
	path := "./testdata/known_good_certCA.pem"

	key, cert, err := NewCA("dummy-test-certificate.local")
	if err != nil {
		t.Fatal(err)
	}
	err = key.Validate()
	if err != nil {
		t.Error(err)
	}
	if os.Getenv(updateSSLVar) == "true" {
		err = writeCert(path, cert)
		if err != nil {
			t.Error(err)
		}
	}
	goodCert, err := readCert(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []*x509.Certificate{cert, goodCert} {
		c.NotBefore = time.Time{}
		c.NotAfter = time.Time{}
		c.PublicKey.(*rsa.PublicKey).N = nil
		c.Raw = nil
		c.RawSubjectPublicKeyInfo = nil
		c.RawTBSCertificate = nil
		c.SerialNumber = nil
		c.Signature = nil
	}

	if !reflect.DeepEqual(cert, goodCert) {
		t.Error("certificates did not match, check test for details")
	}
}

func TestNewCert(t *testing.T) {
	path := "testdata/known_good_cert.pem"

	cn := "dummy-test-certificate.local"
	key, cert, err := NewCert(cn, []string{cn}, []string{cn}, []net.IP{net.ParseIP("192.168.0.1")}, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth | x509.ExtKeyUsageClientAuth}, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	err = key.Validate()
	if err != nil {
		t.Error(err)
	}

	if os.Getenv(updateSSLVar) == "true" {
		err = writeCert(path, cert)
		if err != nil {
			t.Error(err)
		}
	}

	goodCert, err := readCert(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []*x509.Certificate{cert, goodCert} {
		c.NotBefore = time.Time{}
		c.NotAfter = time.Time{}
		c.PublicKey.(*rsa.PublicKey).N = nil
		c.Raw = nil
		c.RawSubjectPublicKeyInfo = nil
		c.RawTBSCertificate = nil
		c.SerialNumber = nil
		c.Signature = nil
	}

	if !reflect.DeepEqual(cert, goodCert) {
		t.Error("certificates did not match, check test for details")
	}
}

func TestSignedCertificate(t *testing.T) {

	cn := "dummy-test-certificate.local"

	signingKey, signingCA, err := NewCA("dummy-test-certificate.local")
	if err != nil {
		t.Error(err)
	}
	_, cert, err := NewCert(cn, []string{cn}, []string{cn}, []net.IP{net.ParseIP("192.168.0.1")}, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth | x509.ExtKeyUsageClientAuth}, signingKey, signingCA, false)
	if err != nil {
		t.Error(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(signingCA)
	keyUsages := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth | x509.ExtKeyUsageClientAuth}
	opts := x509.VerifyOptions{
		DNSName:   cn,
		Roots:     roots,
		KeyUsages: keyUsages,
	}
	if _, err := cert.Verify(opts); err != nil {
		t.Error(err)
	}
}
