package api

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalEmptyStruct(t *testing.T) {
	{
		var c Config
		if err := json.Unmarshal([]byte("{}"), &c); err != nil {
			t.Error(err)
		}
	}
	{
		var c CertKeyPair
		if err := json.Unmarshal([]byte("{}"), &c); err != nil {
			t.Error(err)
		}
	}
}
