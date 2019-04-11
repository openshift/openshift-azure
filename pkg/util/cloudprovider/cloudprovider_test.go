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
useInstanceMetadata: true
cloudProviderBackoff: true
cloudProviderBackoffRetries: 6
cloudProviderBackoffJitter: 1
cloudProviderBackoffDuration: 6
cloudProviderBackoffExponent: 1.5
cloudProviderRateLimit: true
cloudProviderRateLimitQPS: 3
cloudProviderRateLimitBucket: 10`)

	want := Config{
		TenantID:                     "a",
		SubscriptionID:               "b",
		AadClientID:                  "c",
		AadClientSecret:              "d",
		ResourceGroup:                "e",
		Location:                     "f",
		LoadBalancerSku:              "g",
		SecurityGroupName:            "h",
		VMType:                       "i",
		UseInstanceMetadata:          true,
		CloudProviderBackoff:         true,
		CloudProviderBackoffRetries:  6,
		CloudProviderBackoffJitter:   1.0,
		CloudProviderBackoffDuration: 6,
		CloudProviderBackoffExponent: 1.5,
		CloudProviderRateLimit:       true,
		CloudProviderRateLimitQPS:    3.0,
		CloudProviderRateLimitBucket: 10,
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
