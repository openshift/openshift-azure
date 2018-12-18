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
		wantSgn      string
		wantInstance string
		name         string
	}{
		{
			name:         "master",
			role:         api.AgentPoolProfileRoleMaster,
			wantScaleset: "ss-master",
			wantSgn:      "nsg-master",
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
			name:         "compute - thingy",
			role:         api.AgentPoolProfileRoleCompute,
			wantScaleset: "ss-thingy",
			wantSgn:      "nsg-thingy",
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
			if got := GetSecurityGroupName(tt.cs, tt.role); got != tt.wantSgn {
				t.Errorf("GetScalesetName() = %v, want %v", got, tt.wantSgn)
			}
			if got := GetInstanceName(tt.cs, tt.role, tt.instance); got != tt.wantInstance {
				t.Errorf("GetScalesetName() = %v, want %v", got, tt.wantInstance)
			}
		})
	}
}
