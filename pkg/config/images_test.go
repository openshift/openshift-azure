package config

import (
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestNodeImageVersion(t *testing.T) {
	for _, deployOS := range []string{"", "rhel7", "centos7"} {
		cs := api.OpenShiftManagedCluster{
			Properties: &api.Properties{
				OpenShiftVersion: "v3.10",
			},
			Config: &api.Config{},
		}
		selectNodeImage(&cs, deployOS)
		if cs.Config.ImageVersion == "latest" {
			t.Errorf("cs.Config.ImageVersion should not equal latest")
		}
	}
}
