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

// GetPemBlock extracts the requested block out of the data and returns it as a string
func GetPemBlock(data []byte, blockType string) (string, error) {
	for block, remainder := pem.Decode(data); block != nil; block, remainder = pem.Decode(remainder) {
		if block.Type != blockType {
			continue
		}
		switch block.Type {
		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return "", err
			}
			switch key := key.(type) {
			case *rsa.PrivateKey:
				b, err := PrivateKeyAsBytes(key)
				if err != nil {
					return "", err
				}
				return string(b), nil
			default:
				return "", errors.New("found unknown private key type in PKCS#8 wrapping")
			}
		case "CERTIFICATE":
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return "", err
			}
			b, err := CertAsBytes(cert)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	}
	return "", fmt.Errorf("failed to find block %s", blockType)
}

func ParsePrivateKey(b []byte) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode(b)
	if len(rest) > 0 {
		return nil, errors.New("extra data after decoding PEM block")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
	case "PRIVATE KEY":
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			switch key := key.(type) {
			case *rsa.PrivateKey:
				return key, nil
			default:
				return nil, errors.New("found unknown private key type in PKCS#8 wrapping")
			}
		}
	}
	return nil, errors.New(" failed to parse private key")
}
