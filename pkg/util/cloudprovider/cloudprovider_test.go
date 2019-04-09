package cloudprovider

import (
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
)

func TestUnmarshal(t *testing.T) {
	b := []byte(`
tenantId: a
subscriptionId: b
aadClientId: c
aadClientSecret: d
resourceGroup: e
location: f
loadBalancerSku: g
securityGroupName: h
vmType: i
useInstanceMetadata: true`)

	want := Config{
		TenantID:            "a",
		SubscriptionID:      "b",
		AadClientID:         "c",
		AadClientSecret:     "d",
		ResourceGroup:       "e",
		Location:            "f",
		LoadBalancerSku:     "g",
		SecurityGroupName:   "h",
		VMType:              "i",
		UseInstanceMetadata: true,
	}

	var got Config
	err := yaml.Unmarshal(b, &got)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Unmarshal() = %v, want %v", got, want)
	}
}

// Note the Marshal is tested in TestDerivedCloudProviderConf
