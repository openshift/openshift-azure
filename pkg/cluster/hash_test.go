package cluster

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
)

const masterHash = "RmID82LhPjuQbCEdiVa5cGCVEkdLGD6iU6ozX3vxkD0="

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name string
		vmss *compute.VirtualMachineScaleSet
		exp  hash
	}{
		{
			name: "expect a scale set",
			vmss: &compute.VirtualMachineScaleSet{
				Sku:  &compute.Sku{},
				Name: to.StringPtr("ss-master"),
				Type: to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			},
			exp: masterHash,
		},
		{
			name: "hash is invariant with capacity",
			vmss: &compute.VirtualMachineScaleSet{
				Sku: &compute.Sku{
					Capacity: to.Int64Ptr(3),
				},
				Name: to.StringPtr("ss-master"),
				Type: to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			},
			exp: masterHash,
		},
	}

	for _, test := range tests {
		got, err := hashVMSS(test.vmss)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, test.exp, got)
		}
	}
}
