package cluster

import (
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

var masterHash = []byte{0x20, 0x3c, 0xd3, 0x78, 0xd9, 0x84, 0x73, 0x54, 0x1,
	0x13, 0x6, 0x65, 0x1d, 0x7a, 0xaf, 0xd6, 0xaa, 0xcf, 0x7a, 0x4, 0x3f, 0xa,
	0x40, 0x1a, 0x25, 0xab, 0x5, 0xc9, 0xb9, 0xaf, 0xd4, 0x4a}

func TestHashScaleSet(t *testing.T) {
	tests := []struct {
		name string
		app  api.AgentPoolProfile
		exp  []byte
	}{
		{
			name: "hash shouldn't change over time",
			exp:  masterHash,
		},
		{
			name: "hash is invariant with name and count",
			app: api.AgentPoolProfile{
				Name:  "foo",
				Count: 1,
			},
			exp: masterHash,
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
	for _, test := range tests {
		got, err := h.HashScaleSet(&cs, &test.app)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}
		if !reflect.DeepEqual(got, test.exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, test.exp, got)
		}
	}
}
