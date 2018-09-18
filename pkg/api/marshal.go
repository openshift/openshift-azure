package api

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/tls"
)

// makeShadowStruct returns a pointer to a struct identical in type to the one
// passed in, but with certain fields types set to json.RawMessage.
func makeShadowStruct(v reflect.Value) reflect.Value {
	fields := make([]reflect.StructField, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		switch f.Type.String() {
		case "*rsa.PrivateKey", "*v1.Config", "*x509.Certificate":
			f.Type = reflect.TypeOf(json.RawMessage{})
		}
		fields = append(fields, f)
	}

	return reflect.New(reflect.StructOf(fields))
}

// marshalJSON marshals a struct, overriding the marshaling used for x509
// certificates, private keys and kubeconfigs.
func marshalJSON(v reflect.Value) ([]byte, error) {
	shadow := makeShadowStruct(v).Elem()

	// copy data into the shadow struct
	for i := 0; i < v.NumField(); i++ {
		switch v := v.Field(i).Interface().(type) {
		case *rsa.PrivateKey:
			if v == nil {
				continue
			}

			b, err := tls.PrivateKeyAsBytes(v)
			if err != nil {
				return nil, err
			}

			b, err = json.Marshal(b)
			if err != nil {
				return nil, err
			}

			shadow.Field(i).Set(reflect.ValueOf(b))

		case *v1.Config:
			if v == nil {
				continue
			}

			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}

			b, err = json.Marshal(base64.StdEncoding.EncodeToString(b))
			if err != nil {
				return nil, err
			}

			shadow.Field(i).Set(reflect.ValueOf(b))

		case *x509.Certificate:
			if v == nil {
				continue
			}

			b, err := tls.CertAsBytes(v)
			if err != nil {
				return nil, err
			}

			b, err = json.Marshal(b)
			if err != nil {
				return nil, err
			}

			shadow.Field(i).Set(reflect.ValueOf(b))

		default:
			shadow.Field(i).Set(reflect.ValueOf(v))
		}
	}

	return json.Marshal(shadow.Interface())
}

// unmarshalJSON unmarshals a struct, overriding the unmarshaling used for x509
// certificates, private keys and kubeconfigs.
func unmarshalJSON(v reflect.Value, b []byte) error {
	shadow := makeShadowStruct(v)
	err := json.Unmarshal(b, shadow.Interface())
	if err != nil {
		return err
	}
	shadow = shadow.Elem()

	// copy data out of the shadow struct
	for i := 0; i < v.NumField(); i++ {
		switch v.Field(i).Interface().(type) {
		case *rsa.PrivateKey:
			var b []byte
			err = json.Unmarshal(shadow.Field(i).Bytes(), &b)
			if err != nil {
				return err
			}

			key, err := tls.ParsePrivateKey(b)
			if err != nil {
				return err
			}

			v.Field(i).Set(reflect.ValueOf(key))

		case *v1.Config:
			var b []byte
			err = json.Unmarshal(shadow.Field(i).Bytes(), &b)
			if err != nil {
				return err
			}

			var c v1.Config
			err = json.Unmarshal(b, &c)
			if err != nil {
				return err
			}

			v.Field(i).Set(reflect.ValueOf(&c))

		case *x509.Certificate:
			var b []byte
			err = json.Unmarshal(shadow.Field(i).Bytes(), &b)
			if err != nil {
				return err
			}

			cert, err := tls.ParseCert(b)
			if err != nil {
				return err
			}

			v.Field(i).Set(reflect.ValueOf(cert))

		default:
			v.Field(i).Set(shadow.Field(i))
		}
	}

	return nil
}

func (c Config) MarshalJSON() ([]byte, error) {
	return marshalJSON(reflect.ValueOf(c))
}

func (c *Config) UnmarshalJSON(b []byte) error {
	return unmarshalJSON(reflect.ValueOf(c).Elem(), b)
}

func (c CertKeyPair) MarshalJSON() ([]byte, error) {
	return marshalJSON(reflect.ValueOf(c))
}

func (c *CertKeyPair) UnmarshalJSON(b []byte) error {
	return unmarshalJSON(reflect.ValueOf(c).Elem(), b)
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
