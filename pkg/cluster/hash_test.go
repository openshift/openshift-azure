package cluster

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
)

var masterHash = []byte{0x46, 0x62, 0x3, 0xf3, 0x62, 0xe1, 0x3e, 0x3b, 0x90,
	0x6c, 0x21, 0x1d, 0x89, 0x56, 0xb9, 0x70, 0x60, 0x95, 0x12, 0x47, 0x4b,
	0x18, 0x3e, 0xa2, 0x53, 0xaa, 0x33, 0x5f, 0x7b, 0xf1, 0x90, 0x3d}

func TestHashScaleSets(t *testing.T) {
	tests := []struct {
		name string
		vmss *compute.VirtualMachineScaleSet
		exp  []byte
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
