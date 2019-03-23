package names

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

func TestGetHostname(t *testing.T) {
	for _, tt := range []struct {
		app          *api.AgentPoolProfile
		suffix       string
		instance     int64
		wantHostname string
	}{
		{
			app: &api.AgentPoolProfile{
				Name: "master",
				Role: api.AgentPoolProfileRoleMaster,
			},
			wantHostname: "master-000000",
		},
		{
			app: &api.AgentPoolProfile{
				Name: "master",
				Role: api.AgentPoolProfileRoleMaster,
			},
			instance:     10,
			wantHostname: "master-00000a",
		},
		{
			app: &api.AgentPoolProfile{
				Name: "my-compute",
				Role: api.AgentPoolProfileRoleCompute,
			},
			suffix:       "suffix",
			instance:     25294,
			wantHostname: "my-compute-suffix-000jim",
		},
	} {
		hostname := GetHostname(tt.app, tt.suffix, tt.instance)
		if tt.wantHostname != hostname {
			t.Errorf("wanted hostname %s, got %s", tt.wantHostname, hostname)
		}
	}
}

func TestGetScaleSetNameAndInstanceID(t *testing.T) {
	for _, tt := range []struct {
		hostname       string
		wantScaleset   string
		wantInstanceID string
		wantErr        string
	}{
		{
			hostname:       "compute-1234-000000",
			wantScaleset:   "ss-compute-1234",
			wantInstanceID: "0",
		},
		{
			hostname:       "master-00000A",
			wantScaleset:   "ss-master",
			wantInstanceID: "10",
		},
		{
			hostname:       "mycompute-00000a",
			wantScaleset:   "ss-mycompute",
			wantInstanceID: "10",
		},
		{
			hostname: "bad",
			wantErr:  `invalid hostname "bad"`,
		},
		{
			hostname: "bad-bad",
			wantErr:  `invalid hostname "bad-bad"`,
		},
		{
			hostname: "bad-inval!",
			wantErr:  `invalid hostname "bad-inval!"`,
		},
	} {
		scaleset, instanceID, err := GetScaleSetNameAndInstanceID(tt.hostname)
		if (err == nil) != (tt.wantErr == "") || (err != nil && tt.wantErr != err.Error()) {
			t.Errorf("wanted err %v, got %v", tt.wantErr, err)
			continue
		}
		if tt.wantScaleset != scaleset {
			t.Errorf("wanted scaleset %v, got %v", tt.wantScaleset, scaleset)
		}
		if tt.wantInstanceID != instanceID {
			t.Errorf("wanted instanceID %v, got %v", tt.wantInstanceID, instanceID)
		}
	}
}

func TestGetAgentRole(t *testing.T) {
	for _, tt := range []struct {
		hostname      string
		wantAgentrole api.AgentPoolProfileRole
	}{
		{
			hostname:      "compute-1234-000000",
			wantAgentrole: api.AgentPoolProfileRoleCompute,
		},
		{
			hostname:      "mycompute-1234-000000",
			wantAgentrole: api.AgentPoolProfileRoleCompute,
		},
		{
			hostname:      "master-00000A",
			wantAgentrole: api.AgentPoolProfileRoleMaster,
		},
		{
			hostname:      "infra-12345-00000A",
			wantAgentrole: api.AgentPoolProfileRoleInfra,
		},
	} {
		agentrole := GetAgentRole(tt.hostname)
		if tt.wantAgentrole != agentrole {
			t.Errorf("wanted agentrole %v, got %v", tt.wantAgentrole, agentrole)
		}
	}
}
