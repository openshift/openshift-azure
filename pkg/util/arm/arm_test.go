package arm

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

func TestFixupDepends(t *testing.T) {
	subscriptionID := "subscriptionID"
	resourceGroup := "resourceGroup"

	tests := []struct {
		name      string
		resources []interface{}
		expect    []string
	}{
		{
			name: "have deps, but missing resources",
			resources: []interface{}{
				&LoadBalancer{
					LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
							{
								FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
									PublicIPAddress: &network.PublicIPAddress{
										ID: to.StringPtr(resourceid.ResourceID(
											subscriptionID,
											resourceGroup,
											"Microsoft.Network/publicIPAddresses",
											"ip-apiserver",
										)),
									},
								},
							},
						},
					},
					Name: to.StringPtr("lb-apiserver"),
					Type: to.StringPtr("Microsoft.Network/loadBalancers"),
				},
			},
			expect: []string{},
		},
		{
			name: "have deps and dependent resources",
			resources: []interface{}{
				&PublicIPAddress{
					Name: to.StringPtr("ip-apiserver"),
					Type: to.StringPtr("Microsoft.Network/publicIPAddresses"),
				},
				&LoadBalancer{
					LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
						FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
							{
								FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
									PublicIPAddress: &network.PublicIPAddress{
										ID: to.StringPtr(resourceid.ResourceID(
											subscriptionID,
											resourceGroup,
											"Microsoft.Network/publicIPAddresses",
											"ip-apiserver",
										)),
									},
								},
							},
						},
					},
					Name: to.StringPtr("lb-apiserver"),
					Type: to.StringPtr("Microsoft.Network/loadBalancers"),
				},
			},
			expect: []string{"/subscriptions/subscriptionID/resourceGroups/resourceGroup/providers/Microsoft.Network/publicIPAddresses/ip-apiserver"},
		},
	}
	for _, tt := range tests {
		armT := Template{
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

		FixupDepends(subscriptionID, resourceGroup, azuretemplate)
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
