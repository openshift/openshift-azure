package arm

import (
	"bytes"
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
	if !bytes.Equal(expected, b) {
		t.Errorf("derived.CombinedImagePullSecret(): got %q, expected %q", string(b), string(expected))
	}

}
