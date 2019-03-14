package json

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"reflect"

	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/util/tls"
)

// makeShadowStruct returns a pointer to a struct identical in type to the one
// passed in, but with certain fields types set to json.RawMessage.
func makeShadowStruct(v reflect.Value) reflect.Value {
	fields := make([]reflect.StructField, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		switch f.Type.String() {
		case "*rsa.PrivateKey", "*v1.Config", "*x509.Certificate", "[]*x509.Certificate":
			f.Type = reflect.TypeOf(json.RawMessage{})
		}
		fields = append(fields, f)
	}

	return reflect.New(reflect.StructOf(fields))
}

// MarshalJSON marshals a struct, overriding the marshaling used for x509
// certificates, private keys and kubeconfigs.
func MarshalJSON(v reflect.Value) ([]byte, error) {
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

		case []*x509.Certificate:
			if v == nil {
				continue
			}

			b, err := tls.CertChainAsBytes(v)
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

// UnmarshalJSON unmarshals a struct, overriding the unmarshaling used for x509
// certificates, private keys and kubeconfigs.
func UnmarshalJSON(v reflect.Value, b []byte) error {
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
			if len(shadow.Field(i).Bytes()) == 0 {
				continue
			}

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
			if len(shadow.Field(i).Bytes()) == 0 {
				continue
			}

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
			if len(shadow.Field(i).Bytes()) == 0 {
				continue
			}

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

		case []*x509.Certificate:
			if len(shadow.Field(i).Bytes()) == 0 {
				continue
			}

			var b []byte
			err = json.Unmarshal(shadow.Field(i).Bytes(), &b)
			if err != nil {
				return err
			}

			cert, err := tls.ParseCertChain(b)
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
