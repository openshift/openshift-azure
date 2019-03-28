package plugin

import (
	"reflect"

	oajson "github.com/openshift/openshift-azure/pkg/api/json"
)

func (c Config) MarshalJSON() ([]byte, error) {
	return oajson.MarshalJSON(reflect.ValueOf(c))
}

func (c *Config) UnmarshalJSON(b []byte) error {
	return oajson.UnmarshalJSON(reflect.ValueOf(c).Elem(), b)
}

func (c CertKeyPair) MarshalJSON() ([]byte, error) {
	return oajson.MarshalJSON(reflect.ValueOf(c))
}

func (c *CertKeyPair) UnmarshalJSON(b []byte) error {
	return oajson.UnmarshalJSON(reflect.ValueOf(c).Elem(), b)
}
