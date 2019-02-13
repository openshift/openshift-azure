package shared

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestIsScaleOperation(t *testing.T) {
	tests := []struct {
		name     string
		new      *api.OpenShiftManagedCluster
		old      *api.OpenShiftManagedCluster
		expected bool
	}{
		{
			name:     "old and new are the same",
			new:      &api.OpenShiftManagedCluster{},
			old:      &api.OpenShiftManagedCluster{},
			expected: false,
		},
		{
			name: "old and new differ only in the agent pool profile counts",
			new: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 3},
						{Role: api.AgentPoolProfileRoleCompute, Name: "compute", Count: 10},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra", Count: 2},
					},
				},
			},
			old: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 3},
						{Role: api.AgentPoolProfileRoleCompute, Name: "compute", Count: 2},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra", Count: 3},
					},
				},
			},
			expected: true,
		},
		{
			name: "old and new differ in the agent pool profile counts and other parts of config",
			new: &api.OpenShiftManagedCluster{
				Config: api.Config{
					ImageVersion: "311.61.20190204",
				},
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 3},
						{Role: api.AgentPoolProfileRoleCompute, Name: "compute", Count: 5},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra", Count: 2},
					},
				},
			},
			old: &api.OpenShiftManagedCluster{
				Config: api.Config{
					ImageVersion: "311.51.20190104",
				},
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master", Count: 3},
						{Role: api.AgentPoolProfileRoleCompute, Name: "compute", Count: 10},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra", Count: 3},
					},
				},
			},
			expected: false,
		},
		{
			name: "old and new have identical agent pool profiles but differ in other parts of config",
			new: &api.OpenShiftManagedCluster{
				Config: api.Config{
					ImageVersion: "311.61.20190204",
				},
			},
			old: &api.OpenShiftManagedCluster{
				Config: api.Config{
					ImageVersion: "311.51.20190104",
				},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		if got := IsScaleOperation(test.new, test.old); got != test.expected {
			t.Errorf("IsScaleOperation() = %v, want %v", got, test.expected)
		}
	}
}
