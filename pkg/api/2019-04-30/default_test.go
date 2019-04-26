package v20190430

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
)

func sampleManagedCluster() *OpenShiftManagedCluster {
	return &OpenShiftManagedCluster{
		Properties: &Properties{
			MasterPoolProfile: &MasterPoolProfile{
				Count:      to.Int64Ptr(3),
				VMSize:     (*VMSize)(to.StringPtr("Standard_D2s_v3")),
				SubnetCIDR: to.StringPtr("10.0.0.0/24"),
			},
			RouterProfiles: []RouterProfile{
				{
					Name:            to.StringPtr("Properties.RouterProfiles[0].Name"),
					PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
				},
			},
		},
	}
}

func TestDefaults(t *testing.T) {
	tests := []struct {
		name           string
		changeInput    func(*OpenShiftManagedCluster)
		expectedChange func(*OpenShiftManagedCluster)
	}{
		{
			name: "sets all defaults",
			changeInput: func(oc *OpenShiftManagedCluster) {
				oc.Properties = nil
			},
			expectedChange: func(oc *OpenShiftManagedCluster) {
				oc.Properties.MasterPoolProfile = &MasterPoolProfile{
					Count: to.Int64Ptr(3),
				}
				oc.Properties.RouterProfiles = []RouterProfile{
					{
						Name: to.StringPtr("default"),
					},
				}
			},
		},
		{
			name: "sets MasterPoolProfile.Count to 3 when empty",
			changeInput: func(oc *OpenShiftManagedCluster) {
				oc.Properties.MasterPoolProfile = &MasterPoolProfile{
					VMSize:     (*VMSize)(to.StringPtr("Standard_D2s_v3")),
					SubnetCIDR: to.StringPtr("10.0.0.0/24"),
				}
			},
			expectedChange: func(oc *OpenShiftManagedCluster) {
				oc.Properties.MasterPoolProfile = &MasterPoolProfile{
					Count:      to.Int64Ptr(3),
					VMSize:     (*VMSize)(to.StringPtr("Standard_D2s_v3")),
					SubnetCIDR: to.StringPtr("10.0.0.0/24"),
				}
			},
		},
		{
			name: "sets no defaults",
		},
	}

	for _, test := range tests {
		config := sampleManagedCluster()
		if test.changeInput != nil {
			test.changeInput(config)
		}

		expected := sampleManagedCluster()
		if test.expectedChange != nil {
			test.expectedChange(expected)
		}

		setDefaults(config)

		if !reflect.DeepEqual(config, expected) {
			t.Errorf("%s: unexpected diff %s", test.name, deep.Equal(config, expected))
		}
	}
}
