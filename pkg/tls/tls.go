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

type PrivateKey struct {
	rsa.PrivateKey
}

func NewPrivateKey() (*PrivateKey, error) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &PrivateKey{*pk}, nil
}

func (in *PrivateKey) DeepCopy() *PrivateKey {
	out := new(PrivateKey)

	// public part.
	if in.PublicKey.N != nil {
		out.PublicKey.N = deepCopyBigInt(in.PublicKey.N)
		out.PublicKey.E = in.PublicKey.E
	}

	// private exponent
	out.D = deepCopyBigInt(in.D)

	// prime factors of N, has >= 2 elements.
	var outPrimes []*big.Int
	for _, prime := range in.Primes {
		outPrimes = append(outPrimes, deepCopyBigInt(prime))
	}
	out.Primes = outPrimes

	// Precomputed contains precomputed values that speed up private
	// operations, if available.
	out.Precomputed.Dp = deepCopyBigInt(in.Precomputed.Dp)
	out.Precomputed.Dq = deepCopyBigInt(in.Precomputed.Dq)
	out.Precomputed.Qinv = deepCopyBigInt(in.Precomputed.Qinv)
	outCrtValues := make([]rsa.CRTValue, 0)
	for _, crtVal := range in.Precomputed.CRTValues {
		outCrtValues = append(outCrtValues, rsa.CRTValue{
			Exp:   deepCopyBigInt(crtVal.Exp),
			Coeff: deepCopyBigInt(crtVal.Coeff),
			R:     deepCopyBigInt(crtVal.R),
		})
	}
	out.Precomputed.CRTValues = outCrtValues

	return out
}

type Certificate struct {
	x509.Certificate
}

func (in *Certificate) DeepCopy() *Certificate {
	out := new(Certificate)
	if in.Raw != nil {
		out.Raw = make([]byte, len(in.Raw))
		copy(in.Raw, out.Raw)
	}
	if in.RawTBSCertificate != nil {
		out.RawTBSCertificate = make([]byte, len(in.RawTBSCertificate))
		copy(in.RawTBSCertificate, out.RawTBSCertificate)
	}
	if in.RawSubjectPublicKeyInfo != nil {
		out.RawSubjectPublicKeyInfo = make([]byte, len(in.RawSubjectPublicKeyInfo))
		copy(in.RawSubjectPublicKeyInfo, out.RawSubjectPublicKeyInfo)
	}
	if in.RawSubject != nil {
		out.RawSubject = make([]byte, len(in.RawSubject))
		copy(in.RawSubject, out.RawSubject)
	}
	if in.RawIssuer != nil {
		out.RawIssuer = make([]byte, len(in.RawIssuer))
		copy(in.RawIssuer, out.RawIssuer)
	}
	if in.Signature != nil {
		out.Signature = make([]byte, len(in.Signature))
		copy(in.Signature, out.Signature)
	}
	out.SignatureAlgorithm = in.SignatureAlgorithm
	out.PublicKeyAlgorithm = in.PublicKeyAlgorithm
	// TODO: This is a shallow copy, PublicKey is an interface{}
	out.PublicKey = in.PublicKey
	out.Version = in.Version
	out.SerialNumber = deepCopyBigInt(in.SerialNumber)
	// TODO: Shallow copy
	out.Issuer = in.Issuer
	// TODO: Shallow copy
	out.Subject = in.Subject
	out.NotAfter = in.NotBefore
	out.NotAfter = in.NotAfter
	out.KeyUsage = in.KeyUsage
	// TODO: rest
	return out
}

func deepCopyBigInt(in *big.Int) *big.Int {
	var out *big.Int
	if in != nil {
		out = new(big.Int)
		*out = *in
	}
	return out
}

func newCert(key *PrivateKey, template *x509.Certificate, signingkey *PrivateKey, signingcert *Certificate) (*Certificate, error) {
	if signingcert == nil && signingkey == nil {
		// make it self-signed
		signingcert = &Certificate{*template}
		signingkey = key
	}

	b, err := x509.CreateCertificate(rand.Reader, template, &signingcert.Certificate, key.Public(), signingkey)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(b)
	if err != nil {
		return nil, err
	}
	return &Certificate{*cert}, nil
}

func NewCA(cn string) (*PrivateKey, *Certificate, error) {
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
	signingkey *PrivateKey,
	signingcert *Certificate,
	selfSign bool,
) (*PrivateKey, *Certificate, error) {
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
	var cert *Certificate
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
