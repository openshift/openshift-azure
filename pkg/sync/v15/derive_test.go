package sync

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

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

func TestRegistryURL(t *testing.T) {
	cs := api.OpenShiftManagedCluster{
		Config: api.Config{
			Images: api.ImageConfig{
				Format: "quay.io/openshift/origin-${component}:${version}",
			},
		},
	}
	if got := derived.RegistryURL(&cs); got != "quay.io" {
		t.Errorf("derived.RegistryURL() = %v, want %v", got, "quay.io")
	}
}

func TestOpenShiftClientVersion(t *testing.T) {
	cs := api.OpenShiftManagedCluster{
		Config: api.Config{
			Images: api.ImageConfig{
				Console: "registry.access.redhat.com/openshift3/ose-console:v3.11.135",
			},
		},
	}
	want := "3.11.135"
	got, err := derived.OpenShiftClientVersion(&cs)
	if err != nil {
		t.Errorf("derivedType.OpenShiftClientVersion() error = %v", err)
		return
	}
	if got != want {
		t.Errorf("derivedType.OpenShiftClientVersion() = %v, want %v", got, want)
	}

}
