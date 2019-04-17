package arm

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestDerivedMasterLBCNamePrefix(t *testing.T) {
	cs := api.OpenShiftManagedCluster{
		Properties: api.Properties{FQDN: "bar.baz"},
	}
	if got := derived.MasterLBCNamePrefix(&cs); got != "bar" {
		t.Errorf("derived.MasterLBCNamePrefix() = %v, want %v", got, "bar")
	}
}

func TestDerivedRouterLBCNamePrefix(t *testing.T) {
	cs := api.OpenShiftManagedCluster{
		Properties: api.Properties{
			RouterProfiles: []api.RouterProfile{
				{
					FQDN: "one.two.three",
				},
			},
		},
	}
	if got := derived.RouterLBCNamePrefix(&cs); got != "one" {
		t.Errorf("derived.RouterLBCNamePrefix() = %v, want %v", got, "one")
	}
}
