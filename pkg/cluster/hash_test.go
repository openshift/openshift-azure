package cluster

import (
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func TestHashWorkerScaleSet(t *testing.T) {
	tests := []struct {
		name string
		app  api.AgentPoolProfile
	}{
		{
			name: "hash shouldn't change over time",
		},
		{
			name: "hash is invariant with name and count",
			app: api.AgentPoolProfile{
				Name:  "foo",
				Count: 1,
			},
		},
	}

	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []api.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]api.IdentityProvider{{Provider: &api.AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	var cs api.OpenShiftManagedCluster
	populate.Walk(&cs, prepare)

	var h hasher
	var exp []byte
	for _, test := range tests {
		got, err := h.HashWorkerScaleSet(&cs, &test.app)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}
		if exp == nil {
			exp = got
		}
		if !reflect.DeepEqual(got, exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, exp, got)
		}
	}
}
