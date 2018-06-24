package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/jim-minter/azure-helm/pkg/tls"
	"github.com/satori/uuid"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

func (c Config) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	v := reflect.ValueOf(c)
	for i := 0; i < v.NumField(); i++ {
		k := v.Type().Field(i).Name

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

		case ssh.PublicKey:
			m[k] = tls.SSHPublicKeyAsString(v)

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

func (c *Config) UnmarshalJSON(b []byte) error {
	m := map[string]interface{}{}
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		k := v.Type().Field(i).Name

		if _, exists := m[k]; !exists {
			continue
		}

		switch v.Field(i).Type().String() {
		case "net.IP":
			ip := net.ParseIP(m[k].(string))
			v.Field(i).Set(reflect.ValueOf(ip))

		case "*rsa.PrivateKey":
			b, err := base64.StdEncoding.DecodeString(m[k].(string))
			if err != nil {
				return err
			}

			key, err := tls.ParsePrivateKey(b)
			if err != nil {
				return err
			}
			v.Field(i).Set(reflect.ValueOf(key))

		case "*rsa.PublicKey":
			b, err := base64.StdEncoding.DecodeString(m[k].(string))
			if err != nil {
				return err
			}

			key, err := tls.ParsePublicKey(b)
			if err != nil {
				return err
			}
			v.Field(i).Set(reflect.ValueOf(key))

		case "ssh.PublicKey":
			key, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(m[k].(string)))
			if err != nil {
				return err
			}
			if len(rest) > 0 {
				return errors.New("extra data after decoding SSH public key")
			}
			v.Field(i).Set(reflect.ValueOf(key))

		case "uuid.UUID":
			u, err := uuid.FromString(m[k].(string))
			if err != nil {
				return err
			}
			v.Field(i).Set(reflect.ValueOf(u))

		case "*v1.Config":
			b, err := base64.StdEncoding.DecodeString(m[k].(string))
			if err != nil {
				return err
			}

			var c v1.Config
			err = yaml.Unmarshal(b, &c)
			if err != nil {
				return err
			}

			v.Field(i).Set(reflect.ValueOf(&c))

		case "*x509.Certificate":
			b, err := base64.StdEncoding.DecodeString(m[k].(string))
			if err != nil {
				return err
			}

			cert, err := tls.ParseCert(b)
			if err != nil {
				return err
			}
			v.Field(i).Set(reflect.ValueOf(cert))

		case "[]uint8":
			b, err := base64.StdEncoding.DecodeString(m[k].(string))
			if err != nil {
				return err
			}

			v.Field(i).Set(reflect.ValueOf(b))

		default:
			v.Field(i).Set(reflect.ValueOf(m[k]))
		}
	}

	return nil
}
