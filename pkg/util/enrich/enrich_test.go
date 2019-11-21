package enrich

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestPrivateAPIServerIPAddress(t *testing.T) {
	tests := []struct {
		subnet  string
		want    string
		wantErr bool
	}{
		{
			subnet: "10.0.2.0/24",
			want:   "10.0.2.254",
		},
		{
			subnet: "172.0.16.0/28",
			want:   "172.0.16.14",
		},
		{
			subnet: "8.0.2.0/16",
			want:   "8.0.255.254",
		},
		{
			subnet:  "",
			want:    "",
			wantErr: true,
		},
	}
	cs := &api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AgentPoolProfiles: []api.AgentPoolProfile{
				{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
				{Role: api.AgentPoolProfileRoleCompute, Name: "compute"},
				{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.subnet, func(t *testing.T) {
			cs.Properties.AgentPoolProfiles[0].SubnetCIDR = tt.subnet
			cs.Properties.PublicHostname = ""
			err := PrivateAPIServerIPAddress(cs)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLastUsableIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if cs.Properties.PublicHostname != tt.want {
				t.Errorf("getLastUsableIP() = %v, want %v", cs.Properties.PublicHostname, tt.want)
			}
		})
	}
}
