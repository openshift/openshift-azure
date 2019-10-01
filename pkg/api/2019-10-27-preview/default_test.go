package v20191027preview

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/cmp"
)

func sampleManagedCluster() *OpenShiftManagedCluster {
	return &OpenShiftManagedCluster{
		Properties: &Properties{
			MasterPoolProfile: &MasterPoolProfile{
				Count:      to.Int64Ptr(3),
				VMSize:     (*VMSize)(to.StringPtr("Standard_D2s_v3")),
				SubnetCIDR: to.StringPtr("10.0.0.0/24"),
			},
			AgentPoolProfiles: []AgentPoolProfile{
				{
					Name:       to.StringPtr("infra"),
					Count:      to.Int64Ptr(4),
					VMSize:     (*VMSize)(to.StringPtr("Standard_D2s_v3")),
					SubnetCIDR: to.StringPtr("10.0.0.0/24"),
					OSType:     (*OSType)(to.StringPtr("Windows")),
					Role:       (*AgentPoolProfileRole)(to.StringPtr("infra")),
				},
				{
					Name:       to.StringPtr("compute"),
					Count:      to.Int64Ptr(4),
					VMSize:     (*VMSize)(to.StringPtr("Standard_D2s_v3")),
					SubnetCIDR: to.StringPtr("10.0.0.0/24"),
					OSType:     (*OSType)(to.StringPtr("Windows")),
					Role:       (*AgentPoolProfileRole)(to.StringPtr("compute")),
				},
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
				oc.Properties.AgentPoolProfiles = []AgentPoolProfile{
					{
						Name:   to.StringPtr("infra"),
						Count:  to.Int64Ptr(3),
						OSType: (*OSType)(to.StringPtr("Linux")),
						Role:   (*AgentPoolProfileRole)(to.StringPtr("infra")),
					},
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
				oc.Properties.MasterPoolProfile.Count = nil
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
			name: "sets AgentPoolProfile.Count to 3 on infra when empty",
			changeInput: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[0].Count = nil
			},
			expectedChange: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[0].Count = to.Int64Ptr(3)
			},
		},
		{
			name: "sets AgentPoolProfile.OSType to Linux when empty",
			changeInput: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[0].OSType = nil
			},
			expectedChange: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[0].OSType = (*OSType)(to.StringPtr("Linux"))
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
			t.Errorf("%s: unexpected diff %s", test.name, cmp.Diff(config, expected))
		}
	}
}
