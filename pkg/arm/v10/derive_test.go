package arm

import (
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func TestDerivedMasterLBCNamePrefix(t *testing.T) {
	cs := api.OpenShiftManagedCluster{
		Properties: api.Properties{FQDN: "bar.baz"},
	}
	if got := derived.MasterLBCNamePrefix(&cs); got != "bar" {
		t.Errorf("derived.MasterLBCNamePrefix() = %v, want %v", got, "bar")
	}
}

func TestCombinedImagePullSecret(t *testing.T) {
	cfg := api.Config{
		Images: api.ImageConfig{
			ImagePullSecret:       populate.DummyImagePullSecret("registry.redhat.io"),
			GenevaImagePullSecret: populate.DummyImagePullSecret("osarpint.azurecr.io"),
		},
	}
	expected := []byte("{\"auths\":{\"osarpint.azurecr.io\":{\"auth\":\"dGVzdDp0ZXN0Cg==\"},\"registry.redhat.io\":{\"auth\":\"dGVzdDp0ZXN0Cg==\"}}}")

	b, err := derived.CombinedImagePullSecret(&cfg)
	if err != nil {
		t.Errorf("derived.CombinedImagePullSecret() error %v", err)
	}
	if !reflect.DeepEqual(expected, b) {
		t.Errorf("derived.CombinedImagePullSecret() = lenght \"%v\", want \"%v\"", len(b), len(expected))
	}

}
