package cluster

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
)

func TestFilterOldVMs(t *testing.T) {
	tests := []struct {
		name     string
		vms      []compute.VirtualMachineScaleSetVM
		blob     map[instanceName]hash
		ssHashes map[scalesetName]hash
		exp      []compute.VirtualMachineScaleSetVM
	}{
		{
			name: "one updated, two old vms",
			vms: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master_0"),
				},
				{
					Name: to.StringPtr("ss-master_1"),
				},
				{
					Name: to.StringPtr("ss-master_2"),
				},
			},
			blob: map[instanceName]hash{
				"ss-master_0": "newhash",
				"ss-master_1": "oldhash",
				"ss-master_2": "oldhash",
			},
			ssHashes: map[scalesetName]hash{
				"ss-master": "newhash",
			},
			exp: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master_1"),
				},
				{
					Name: to.StringPtr("ss-master_2"),
				},
			},
		},
		{
			name: "all updated",
			vms: []compute.VirtualMachineScaleSetVM{
				{
					Name: to.StringPtr("ss-master_0"),
				},
				{
					Name: to.StringPtr("ss-master_1"),
				},
				{
					Name: to.StringPtr("ss-master_2"),
				},
			},
			blob: map[instanceName]hash{
				"ss-master_0": "newhash",
				"ss-master_1": "newhash",
				"ss-master_2": "newhash",
			},
			ssHashes: map[scalesetName]hash{
				"ss-master": "newhash",
			},
			exp: nil,
		},
	}

	u := &simpleUpgrader{
		log: logrus.NewEntry(logrus.StandardLogger()),
	}
	for _, test := range tests {
		t.Logf("running scenario %q", test.name)
		got := u.filterOldVMs(test.vms, test.blob, test.ssHashes)
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("expected vms:\n%#v\ngot:\n%#v", test.exp, got)
		}
	}
}
