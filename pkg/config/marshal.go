package config

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/satori/uuid"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/tls"
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

		case *v1.Config:
			b, err := yaml.Marshal(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case []byte:
			m[k] = base64.StdEncoding.EncodeToString(v)
		case *CertificateConfig:
			bytes, err := yaml.Marshal(v)
			if err != nil {
				return nil, err
			}
			m[k] = bytes
		default:
			m[k] = v
		}
	}
	return json.Marshal(m)
}

func (c *Config) UnmarshalJSON(b []byte) error {
	d := json.NewDecoder(bytes.NewBuffer(b))
	d.UseNumber()

	m := map[string]interface{}{}
	err := d.Decode(&m)
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

		case "int":
			ii, err := m[k].(json.Number).Int64()
			if err != nil {
				return err
			}

			v.Field(i).Set(reflect.ValueOf(int(ii)))

		case "config.CertificateConfig":
			// I don't know if this is the most efficient way to do this
			data, err := yaml.Marshal(m[k])
			if err != nil {
				return err
			}

			var c CertificateConfig
			err = yaml.Unmarshal(data, &c)

			if err != nil {
				return err
			}

			v.Field(i).Set(reflect.ValueOf(c))
		default:
			v.Field(i).Set(reflect.ValueOf(m[k]))
		}
	}

	return nil
}

func (c CertKeyPair) MarshalJSON() ([]byte, error) {
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
		default:
			m[k] = v
		}
	}
	return json.Marshal(m)
}

func (c *CertKeyPair) UnmarshalJSON(b []byte) error {
	d := json.NewDecoder(bytes.NewBuffer(b))
	d.UseNumber()

	m := map[string]interface{}{}
	err := d.Decode(&m)
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

		default:
			v.Field(i).Set(reflect.ValueOf(m[k]))
		}
	}

	return nil
}
