package tls

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"

	"golang.org/x/crypto/ssh"
)

func CertAsBytes(cert *Certificate) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate.Raw})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func PrivateKeyAsBytes(key *PrivateKey) ([]byte, error) {
	buf := &bytes.Buffer{}

	err := pem.Encode(buf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(&key.PrivateKey)})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func PublicKeyAsBytes(key *rsa.PublicKey) ([]byte, error) {
	buf := &bytes.Buffer{}

	b, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}

	err = pem.Encode(buf, &pem.Block{Type: "PUBLIC KEY", Bytes: b})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func SSHPublicKeyAsString(key *rsa.PublicKey) (string, error) {
	sshkey, err := ssh.NewPublicKey(key)
	if err != nil {
		return "", err
	}

	return sshkey.Type() + " " + base64.StdEncoding.EncodeToString(sshkey.Marshal()), nil
}

func ParseCert(b []byte) (*x509.Certificate, error) {
	block, rest := pem.Decode(b)
	if len(rest) > 0 {
		return nil, errors.New("extra data after decoding PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, err
}

func ParsePrivateKey(b []byte) (*PrivateKey, error) {
	block, rest := pem.Decode(b)
	if len(rest) > 0 {
		return nil, errors.New("extra data after decoding PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &PrivateKey{*key}, nil
}
