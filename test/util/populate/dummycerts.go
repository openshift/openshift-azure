package populate

import (
	"crypto/rsa"
	"crypto/x509"
	"reflect"

	"github.com/openshift/openshift-azure/test/util/tls"
)

func DummyCertsAndKeys(v interface{}) {
	var walk func(v reflect.Value)

	walk = func(v reflect.Value) {
		switch v.Interface().(type) {
		case *rsa.PrivateKey:
			v.Set(reflect.ValueOf(tls.DummyPrivateKey))
			return

		case *x509.Certificate:
			v.Set(reflect.ValueOf(tls.DummyCertificate))
			return

		case []*x509.Certificate:
			v.Set(reflect.ValueOf([]*x509.Certificate{tls.DummyCertificate, tls.DummyCertificate}))
			return
		}

		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			walk(v.Elem())

		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		}
	}

	walk(reflect.ValueOf(v))
}
