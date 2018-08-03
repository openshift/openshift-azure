package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"
)

func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func newCert(key *rsa.PrivateKey, template *x509.Certificate, signingkey *rsa.PrivateKey, signingcert *x509.Certificate) (*x509.Certificate, error) {
	if signingcert == nil && signingkey == nil {
		// make it self-signed
		signingcert = template
		signingkey = key
	}

	b, err := x509.CreateCertificate(rand.Reader, template, signingcert, key.Public(), signingkey)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(b)
}

func NewCA(cn string) (*rsa.PrivateKey, *x509.Certificate, error) {
	now := time.Now()

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             now,
		NotAfter:              now.AddDate(5, 0, 0),
		Subject:               pkix.Name{CommonName: cn},
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		IsCA:                  true,
	}

	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	cert, err := newCert(key, template, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	return key, cert, nil
}

func NewCert(
	cn string,
	organization []string,
	dnsNames []string,
	ipAddresses []net.IP,
	extKeyUsage []x509.ExtKeyUsage,
	signingkey *rsa.PrivateKey,
	signingcert *x509.Certificate,
	selfSign bool,
) (*rsa.PrivateKey, *x509.Certificate, error) {
	now := time.Now()

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             now,
		NotAfter:              now.AddDate(2, 0, 0),
		Subject:               pkix.Name{CommonName: cn, Organization: organization},
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           extKeyUsage,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	var cert *x509.Certificate
	if selfSign {
		cert, err = newCert(key, template, nil, nil)
	} else {
		cert, err = newCert(key, template, signingkey, signingcert)
	}
	if err != nil {
		return nil, nil, err
	}

	return key, cert, nil
}
