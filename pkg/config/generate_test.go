package config

import (
	"fmt"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestGenerateClusterID(t *testing.T) {
	props := &api.Properties{
		MasterProfile: &api.MasterProfile{
			FQDN: "foo",
		},
	}
	id := generateClusterId(props)
	if id == "" {
		t.Errorf("cluster id not generated from fully formed profile")
	}
}

func TestGenerate(t *testing.T) {
	// TODO make more tests!
	tests := map[string]struct {
		setterFunc func(*api.ContainerService)
		testFunc   func(*api.ContainerService) error
	}{
		"test ID is generated": {
			testFunc: func(c *api.ContainerService) error {
				if c.ID == "" {
					return fmt.Errorf("failed to generate ID")
				}
				return nil
			},
		},
	}

	for name, test := range tests {
		cs := createTestContainerService()
		if test.setterFunc != nil {
			test.setterFunc(cs)
		}

		if err := Generate(cs); err != nil {
			t.Errorf("%s had error generating: %#v", name, err)
			continue
		}

		if err := test.testFunc(cs); err != nil {
			t.Errorf("%s returned error: %#v", name, err)
		}
	}
}

func createTestContainerService() *api.ContainerService {
	return &api.ContainerService{
		Properties: &api.Properties{
			OrchestratorProfile: &api.OrchestratorProfile{
				OpenShiftConfig: &api.OpenShiftConfig{
					PublicHostname: "foo.bar",
					RouterProfiles: []api.OpenShiftRouterProfile{
						{
							PublicSubdomain: "bar",
						},
					},
				},
				OrchestratorVersion: "X.X",
			},
			MasterProfile: &api.MasterProfile{
				FQDN: "foo.bar",
			},
			AzProfile: &api.AzProfile{},
		},
		Config: &api.Config{},
	}
}
