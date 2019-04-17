package sync

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

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
