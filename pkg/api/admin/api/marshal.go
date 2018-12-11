package api

import (
	"encoding/json"
	"fmt"
	"reflect"

	oajson "github.com/openshift/openshift-azure/pkg/api/json"
)

func (c Config) MarshalJSON() ([]byte, error) {
	return oajson.MarshalJSON(reflect.ValueOf(c))
}

func (c *Config) UnmarshalJSON(b []byte) error {
	return oajson.UnmarshalJSON(reflect.ValueOf(c).Elem(), b)
}

func (c Certificate) MarshalJSON() ([]byte, error) {
	return oajson.MarshalJSON(reflect.ValueOf(c))
}

func (c *Certificate) UnmarshalJSON(b []byte) error {
	return oajson.UnmarshalJSON(reflect.ValueOf(c).Elem(), b)
}

func (ip *IdentityProvider) UnmarshalJSON(b []byte) error {
	dummy := struct {
		Name     *string         `json:"name,omitempty"`
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
