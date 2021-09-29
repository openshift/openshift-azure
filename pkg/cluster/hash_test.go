package cluster

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/sync"
	"github.com/openshift/openshift-azure/test/util/populate"
)

// These tests would be stronger were we to populate all the hash inputs rather
// than leaving most of them at the zero value.  However this approach is
// hampered by the fact that we do not version the internal representation. Were
// we not to use zero values, future refactors of the internal representation
// could move fields around and easily provoke false positives here.

func TestHashScaleSetStability(t *testing.T) {
	// IMPORTANT: hashes are free to change for the version under development,
	// but should be fixed thereafter.  If this test is failing against a
	// released version, do not change the hash here, but fix the underlying
	// problem, otherwise it will result in unwanted rotations in production.

	tests := map[string][]struct {
		role         api.AgentPoolProfileRole
		expectedHash string
	}{
		"v20.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "9c44ae2529186856b014569eadd07e166c81bbfa348c117d84f66efc827e6ef4",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "b59dd4459ddf9775bebbf4547b180dcb4fe0a8b674fccc579bb56c840d423a1e",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "d4b9e78ba42cdb4e178ab806013ad05651993fd30cb396521a88a242211d5621",
			},
		},
		"v21.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "dd81e18c638ac05be1b54a630590428f5ecaf2494bf9babbb75c1ef3b64e8a0b",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "cc749481463d0cba36b2d734a8315ab062f658e007df999a122e8605e2b74071",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "a098b58bd1e671dbe6572d786896da173ce47a5c0b28162a8eba46e12fce8cd4",
			},
		},
		"v22.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "c1be7e53f5b7a706bb0331de2c02ab3fb0e9a68e9b1298f79467b99e52deba92",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "cc749481463d0cba36b2d734a8315ab062f658e007df999a122e8605e2b74071",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "a098b58bd1e671dbe6572d786896da173ce47a5c0b28162a8eba46e12fce8cd4",
			},
		},
	}

	// check we're testing all versions in our pluginconfig
	b, err := ioutil.ReadFile("../../pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var template *pluginapi.Config
	err = yaml.Unmarshal(b, &template)
	if err != nil {
		t.Fatal(err)
	}

	for version := range template.Versions {
		if _, found := tests[version]; !found && version != template.PluginVersion {
			t.Errorf("update tests to include version %s", version)
		}
	}

	cs := &api.OpenShiftManagedCluster{
		ID: "subscriptions/foo/resourceGroups/bar/providers/baz/qux/quz",
		Properties: api.Properties{
			RouterProfiles: []api.RouterProfile{
				{},
			},
			AgentPoolProfiles: []api.AgentPoolProfile{
				{
					Role: api.AgentPoolProfileRoleMaster,
				},
				{
					Role: api.AgentPoolProfileRoleInfra,
				},
				{
					Role:   api.AgentPoolProfileRoleCompute,
					VMSize: api.StandardD2sV3,
				},
			},
			AuthProfile: api.AuthProfile{
				IdentityProviders: []api.IdentityProvider{
					{
						Provider: &api.AADIdentityProvider{},
					},
				},
			},
		},
	}
	populate.DummyCertsAndKeys(cs)

	for version, tests := range tests {

		cs.Config.Images.ImagePullSecret = populate.DummyImagePullSecret("registry.redhat.io")
		cs.Config.Images.GenevaImagePullSecret = populate.DummyImagePullSecret("osarpint.azurecr.io")

		for _, tt := range tests {
			cs.Config.PluginVersion = version

			arm, err := arm.New(context.Background(), nil, cs, api.TestConfig{})
			if err != nil {
				t.Fatal(err)
			}

			hasher := Hash{
				StartupFactory: startup.New,
				Arm:            arm,
			}

			b, err := hasher.HashScaleSet(cs, &api.AgentPoolProfile{Role: tt.role})
			if err != nil {
				t.Fatal(err)
			}

			h := hex.EncodeToString(b)
			if h != tt.expectedHash {
				t.Errorf("%s: %s: hash changed to %s", version, tt.role, h)
			}
		}
	}
}

func TestHashSyncPodStability(t *testing.T) {
	// IMPORTANT: hashes are free to change for the version under development,
	// but should be fixed thereafter.  If this test is failing against a
	// released version, do not change the hash here, but fix the underlying
	// problem, otherwise it will result in unwanted rotations in production.

	tests := map[string]struct {
		expectedHash string
	}{
		"v20.0": {
			// this value should not change
			expectedHash: "2a0838ac55163113989f9c7c39e50bb3c743cd6ecff1408a82e1001e4c152930",
		},
		"v21.0": {
			// this value should not change
			expectedHash: "2a0838ac55163113989f9c7c39e50bb3c743cd6ecff1408a82e1001e4c152930",
		},
		"v22.0": {
			// this value should not change
			expectedHash: "2a0838ac55163113989f9c7c39e50bb3c743cd6ecff1408a82e1001e4c152930",
		},
	}

	// check we're testing all versions in our pluginconfig
	b, err := ioutil.ReadFile("../../pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var template *pluginapi.Config
	err = yaml.Unmarshal(b, &template)
	if err != nil {
		t.Fatal(err)
	}

	for version := range template.Versions {
		if _, found := tests[version]; !found && version != template.PluginVersion {
			t.Errorf("update tests to include version %s", version)
		}
	}

	cs := &api.OpenShiftManagedCluster{
		ID: "subscriptions/foo/resourceGroups/bar/providers/baz/qux/quz",
		Properties: api.Properties{
			RouterProfiles: []api.RouterProfile{
				{},
			},
			AuthProfile: api.AuthProfile{
				IdentityProviders: []api.IdentityProvider{
					{
						Provider: &api.AADIdentityProvider{},
					},
				},
			},
		},
		Config: api.Config{
			ImageVersion: "311.0.0",
			Images: api.ImageConfig{
				AlertManager:             ":",
				ConfigReloader:           ":",
				Grafana:                  ":",
				KubeRbacProxy:            ":",
				KubeStateMetrics:         ":",
				NodeExporter:             ":",
				OAuthProxy:               ":",
				Prometheus:               ":",
				PrometheusConfigReloader: ":",
				PrometheusOperator:       ":",
			},
		},
	}
	populate.DummyCertsAndKeys(cs)

	for version, tt := range tests {
		cs.Config.PluginVersion = version
		// needed by derived.OpenShiftClientVersion()
		cs.Config.Images.Console = "foo:v1.2.3"

		s, err := sync.New(nil, cs, false)
		if err != nil {
			t.Fatal(err)
		}

		b, err := s.Hash()
		if err != nil {
			t.Fatal(err)
		}

		h := hex.EncodeToString(b)
		if h != tt.expectedHash {
			t.Errorf("%s: hash changed to %s", version, h)
		}
	}
}
