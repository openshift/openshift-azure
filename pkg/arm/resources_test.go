package arm

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
)

func TestFixupDepends(t *testing.T) {
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AzProfile: api.AzProfile{
				SubscriptionID: "sess",
				ResourceGroup:  "rg",
			},
		},
		Location: "here",
	}
	tests := []struct {
		name      string
		resources []interface{}
		expect    []string
	}{
		{
			name: "have deps, but missing resources",
			resources: []interface{}{
				lbAPIServer(cs, api.TestConfig{}),
			},
			expect: []string{},
		},
		{
			name: "have deps and dependent resources",
			resources: []interface{}{
				ipAPIServer(cs),
				lbAPIServer(cs, api.TestConfig{}),
			},
			expect: []string{"/subscriptions/sess/resourceGroups/rg/providers/Microsoft.Network/publicIPAddresses/ip-apiserver"},
		},
	}
	for _, tt := range tests {
		armT := armTemplate{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Resources:      tt.resources,
		}
		b, err := json.Marshal(armT)
		if err != nil {
			t.Fatal(err)
		}

		var azuretemplate map[string]interface{}
		err = json.Unmarshal(b, &azuretemplate)
		if err != nil {
			t.Fatal(err)
		}

		fixupDepends(&cs.Properties.AzProfile, azuretemplate)
		res := jsonpath.MustCompile("$.resources[?(@.name='lb-apiserver')]").MustGetObject(azuretemplate)
		deps, found := res["dependsOn"].([]string)
		if !found && len(tt.expect) > 0 {
			t.Fatalf("expected %v, got %v", tt.expect, deps)
		}
		if found && !reflect.DeepEqual(deps, tt.expect) {
			t.Fatalf("expected %v, got %v", tt.expect, deps)
		}
	}
}
