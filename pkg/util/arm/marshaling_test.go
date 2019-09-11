package arm

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
)

func TestMarshaler(t *testing.T) {
	tests := []struct {
		name     string
		resource interface{}
		// Field: JSON path mapping
		expectedJSONFields map[string]string
	}{
		{
			name: "VirtualNetwork",
			resource: VirtualNetwork{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "PublicIPAddress",
			resource: PublicIPAddress{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "LoadBalancer",
			resource: LoadBalancer{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "Account",
			resource: Account{
				Sku: &storage.Sku{
					Name: storage.StandardLRS,
				},
				Kind:     storage.Storage,
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Sku":      "$.sku",
				"Kind":     "$.kind",
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "SecurityGroup",
			resource: SecurityGroup{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "VirtualMachineScaleSet",
			resource: VirtualMachineScaleSet{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "VirtualMachine",
			resource: VirtualMachine{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Compute/VirtualMachines"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "VirtualMachineExtension",
			resource: VirtualMachineExtension{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Compute/virtualMachines/extensions"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
		{
			name: "Interface",
			resource: Interface{
				Name:     to.StringPtr("foo"),
				Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
				Location: to.StringPtr("westus"),
			},
			expectedJSONFields: map[string]string{
				"Name":     "$.name",
				"Type":     "$.type",
				"Location": "$.location",
			},
		},
	}

	for _, test := range tests {
		data, err := json.Marshal(test.resource)
		if err != nil {
			t.Errorf("Unexpected marshaling error: %s", err.Error())
		}

		var dataMap map[string]interface{}
		err = json.Unmarshal(data, &dataMap)
		if err != nil {
			t.Errorf("Unexpected unmarshaling error: %s", err.Error())
		}

		for fieldName, jsonPath := range test.expectedJSONFields {
			if len(jsonpath.MustCompile(jsonPath).Get(dataMap)) != 1 {
				t.Errorf(
					"%s: didn't manage to find field %s in json: %s",
					test.name, fieldName, data,
				)
			}
		}
	}
}
