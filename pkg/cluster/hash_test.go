package cluster

import (
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func TestHashScaleSet(t *testing.T) {
	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []api.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]api.IdentityProvider{{Provider: &api.AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	var cs api.OpenShiftManagedCluster
	populate.Walk(&cs, prepare)
	cs.Properties.AgentPoolProfiles = []api.AgentPoolProfile{
		{
			Role: api.AgentPoolProfileRoleMaster,
		},
		{
			Role:   api.AgentPoolProfileRoleCompute,
			VMSize: api.StandardD2sV3,
		},
	}

	for _, role := range []api.AgentPoolProfileRole{api.AgentPoolProfileRoleMaster, api.AgentPoolProfileRoleCompute} {
		var h hasher
		baseline, err := h.HashScaleSet(&cs, &api.AgentPoolProfile{
			Role: role,
		})
		if err != nil {
			t.Errorf("%s: unexpected error: %v", role, err)
		}
		second, err := h.HashScaleSet(&cs, &api.AgentPoolProfile{
			Name:  "foo",
			Role:  role,
			Count: 1,
		})
		if err != nil {
			t.Errorf("%s: unexpected error: %v", role, err)
		}
		if !reflect.DeepEqual(baseline, second) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", role, baseline, second)
		}
	}
}
