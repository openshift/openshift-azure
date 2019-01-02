package config

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestGetNames(t *testing.T) {
	tests := []struct {
		name         string
		app          *api.AgentPoolProfile
		suffix       string
		wantScaleset string
	}{
		{
			name: "master",
			app: &api.AgentPoolProfile{
				Role: api.AgentPoolProfileRoleMaster,
			},
			suffix:       "0",
			wantScaleset: "ss-master",
		},
		{
			name: "thingy",
			app: &api.AgentPoolProfile{
				Role: api.AgentPoolProfileRoleCompute,
				Name: "thingy",
			},
			suffix:       "foo",
			wantScaleset: "ss-thingy-foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetScalesetName(tt.app, tt.suffix); got != tt.wantScaleset {
				t.Errorf("GetScalesetName() = %v, want %v", got, tt.wantScaleset)
			}
		})
	}
}

func TestGetMasterInstanceName(t *testing.T) {
	if n := GetMasterInstanceName(0); n != "ss-master_0" {
		t.Error(n)
	}
	if n := GetMasterInstanceName(10); n != "ss-master_10" {
		t.Error(n)
	}
}
