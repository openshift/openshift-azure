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
loadBalancerSku: l
location: f
securityGroupName: g
primaryScaleSetName: h
vmType: i`)

	want := Config{
		TenantID:            "a",
		SubscriptionID:      "b",
		AadClientID:         "c",
		AadClientSecret:     "d",
		ResourceGroup:       "e",
		LoadBalancerSku:     "l",
		Location:            "f",
		SecurityGroupName:   "g",
		PrimaryScaleSetName: "h",
		VMType:              "i",
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
