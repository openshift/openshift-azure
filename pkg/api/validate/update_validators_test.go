package validate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestValidateUpdateContainerService(t *testing.T) {
	var twoAgents = []byte(`---
location: eastus
name: openshift
properties:
  publicHostname: donotchange
  routerProfiles:
  - name: default
    publicSubdomain: test.example.com
  agentPoolProfiles:
  - name: infra
    role: infra
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: mycompute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
`)

	tests := map[string]struct {
		newAgentCount    int64
		oldAgentCount    int64
		newHostNameValue string
		wantErrs         []error
	}{
		"good-2": {
			newAgentCount: 1,
			oldAgentCount: 1,
		},
		"good-2-count-not-important": {
			newAgentCount: 5,
			oldAgentCount: 2,
		},
		"bad-field-change": {
			newAgentCount:    1,
			oldAgentCount:    1,
			newHostNameValue: "different",
			wantErrs:         []error{errors.New("invalid change [Properties.PublicHostname: different != donotchange]")},
		},
	}

	for name, tt := range tests {
		var newCs *api.OpenShiftManagedCluster
		var oldCs *api.OpenShiftManagedCluster

		err := yaml.Unmarshal(twoAgents, &oldCs)
		if err != nil {
			t.Fatal(err)
		}
		newCs = oldCs.DeepCopy()

		for i := range oldCs.Properties.AgentPoolProfiles {
			oldCs.Properties.AgentPoolProfiles[i].Count = tt.oldAgentCount
		}
		for i := range newCs.Properties.AgentPoolProfiles {
			newCs.Properties.AgentPoolProfiles[i].Count = tt.newAgentCount
		}
		if tt.newHostNameValue != "" {
			newCs.Properties.PublicHostname = tt.newHostNameValue
		}

		gotErrs := validateUpdateContainerService(newCs, oldCs)
		if !reflect.DeepEqual(gotErrs, tt.wantErrs) {
			t.Errorf("validateUpdateContainerService:%s() = %v, want %v", name, gotErrs, tt.wantErrs)
		}
	}
}
