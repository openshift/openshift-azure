package tls

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func CertAsBytes(cert *x509.Certificate) (b []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("%v", r)
		}
	}()

	buf := &bytes.Buffer{}

	err = pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func PrivateKeyAsBytes(key *rsa.PrivateKey) (b []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("%v", r)
		}
	}()

	buf := &bytes.Buffer{}

	err = pem.Encode(buf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func PublicKeyAsBytes(key *rsa.PublicKey) (b []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			b, err = nil, fmt.Errorf("%v", r)
		}
	}()

	buf := &bytes.Buffer{}

	b, err = x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}

	err = pem.Encode(buf, &pem.Block{Type: "PUBLIC KEY", Bytes: b})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func SSHPublicKeyAsString(key *rsa.PublicKey) (s string, err error) {
	defer func() {
		if r := recover(); r != nil {
			s, err = "", fmt.Errorf("%v", r)
		}
	}()

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

func ParsePrivateKey(b []byte) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode(b)
	if len(rest) > 0 {
		return nil, errors.New("extra data after decoding PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key, nil
}
