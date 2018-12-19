package config

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestGetNames(t *testing.T) {
	tests := []struct {
		cs           *api.OpenShiftManagedCluster
		role         api.AgentPoolProfileRole
		instance     int
		wantScaleset string
		wantInstance string
		name         string
	}{
		{
			name:         "master",
			role:         api.AgentPoolProfileRoleMaster,
			wantScaleset: "ss-master",
			wantInstance: "ss-master_2",
			instance:     2,
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
						{Role: api.AgentPoolProfileRoleCompute, Name: "thingy"},
					},
				},
			},
		},
		{
			name:         "thingy",
			role:         api.AgentPoolProfileRoleCompute,
			wantScaleset: "ss-thingy",
			wantInstance: "ss-thingy_3",
			instance:     3,
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
						{Role: api.AgentPoolProfileRoleCompute, Name: "thingy"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetScalesetName(tt.cs, tt.role); got != tt.wantScaleset {
				t.Errorf("GetScalesetName() = %v, want %v", got, tt.wantScaleset)
			}
			if got := GetInstanceName(tt.name, tt.instance); got != tt.wantInstance {
				t.Errorf("GetInstanceName() = %v, want %v", got, tt.wantInstance)
			}
		})
	}
}
