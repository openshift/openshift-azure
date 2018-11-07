package cluster

import (
	"reflect"
	"testing"
)

const masterHash = "RmID82LhPjuQbCEdiVa5cGCVEkdLGD6iU6ozX3vxkD0="

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name string
		t    map[string]interface{}
		exp  map[scalesetName]hash
	}{
		{
			name: "expect a scale set",
			t: map[string]interface{}{
				"schema": "schemaversion",
				"resources": []interface{}{
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"dependsOn": []interface{}{
							"[resourceId('Microsoft.Network/virtualNetworks', 'vnet')]",
							"[resourceId('Microsoft.Network/networkSecurityGroups', 'nsg-master')]",
						},
						"sku": map[string]interface{}{
							"capacity": "3",
						},
						"name": "ss-master",
					},
					map[string]interface{}{
						"type": "Microsoft.Storage/storageAccounts",
						"name": "dsdgskjgjner",
					},
				},
			},
			exp: map[scalesetName]hash{
				"ss-master": masterHash,
			},
		},
		{
			name: "expect three scale sets",
			t: map[string]interface{}{
				"schema": "schemaversion",
				"resources": []interface{}{
					map[string]interface{}{

						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"dependsOn": []interface{}{
							"[resourceId('Microsoft.Network/virtualNetworks', 'vnet')]",
						},
						"sku": map[string]interface{}{
							"capacity": "2",
						},
						"name": "ss-master",
					},
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"sku": map[string]interface{}{
							"capacity": "2",
						},
						"name": "ss-infra",
					},
					map[string]interface{}{
						"type": "Microsoft.Compute/virtualMachineScaleSets",
						"sku": map[string]interface{}{
							"capacity": "1",
						},
						"name": "ss-compute",
					},
					map[string]interface{}{
						"type": "Microsoft.Storage/storageAccounts",
						"name": "dsdgskjgjner",
					},
				},
			},
			exp: map[scalesetName]hash{
				"ss-master":  masterHash,
				"ss-infra":   "aqOO0n4n/nx5onYVUEwoW3s/GCnFoEZIZBowvhaHD6c=",
				"ss-compute": "iWDo277FXQHmvzHj5z1l4o+L/hoRvVSzTGroojwA2ZU=",
			},
		},
	}

	for _, test := range tests {
		got, err := hashScaleSets(test.t)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, test.exp, got)
		}
	}
}
