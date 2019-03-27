package v20190430

import (
	"encoding/json"
	"fmt"
)

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
