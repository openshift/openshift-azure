package api

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/satori/go.uuid"
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
			if v == nil {
				continue
			}

			b, err := tls.CertAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case *rsa.PrivateKey:
			if v == nil {
				continue
			}

			b, err := tls.PrivateKeyAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case *v1.Config:
			if v == nil {
				continue
			}

			b, err := yaml.Marshal(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case []byte:
			if v == nil {
				continue
			}

			m[k] = base64.StdEncoding.EncodeToString(v)

		case *CertificateConfig:
			if v == nil {
				continue
			}

			bytes, err := yaml.Marshal(v)
			if err != nil {
				return nil, err
			}
			m[k] = bytes

		default:
			if v == nil {
				continue
			}

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

		case "api.CertificateConfig":
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
			if v == nil {
				continue
			}

			b, err := tls.CertAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		case *rsa.PrivateKey:
			if v == nil {
				continue
			}

			b, err := tls.PrivateKeyAsBytes(v)
			if err != nil {
				return nil, err
			}
			m[k] = base64.StdEncoding.EncodeToString(b)

		default:
			if v == nil {
				continue
			}

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

func (ip *IdentityProvider) UnmarshalJSON(b []byte) error {
	dummy := struct {
		Name     string          `json:"name,omitempty"`
		Provider json.RawMessage `json:"provider,omityempty"`
	}{}
	err := json.Unmarshal(b, &dummy)
	if err != nil {
		return err
	}
	// peek inside to find out type
	m := map[string]interface{}{}
	err = json.Unmarshal(dummy.Provider, &m)
	if err != nil {
		return err
	}

	switch m["kind"] {
	case "AADIdentityProvider":
		ip.Provider = &AADIdentityProvider{}
		//unmarshal to the right type
		err = json.Unmarshal(dummy.Provider, &ip.Provider)
		if err != nil {
			return err
		}
		ip.Name = dummy.Name
	default:
		return fmt.Errorf("unsupported identity provider kind %q", m["kind"])
	}

	return nil
}
