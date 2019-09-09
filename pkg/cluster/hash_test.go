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
		"v5.1": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "4d4d1c053cba07da9641a817d7ff5712a40d6f43ba10306dc802a4d3d8b71932",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "d954d5572f9a67de96bbaf97fc89c62c5625812890a6cc39debd9c6fb7e0dd47",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "c43d513804031e9b14f076a53c5f02b8ae121926ca7e2f33cbaad1b22cde89b8",
			},
		},
		"v5.2": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "4d4d1c053cba07da9641a817d7ff5712a40d6f43ba10306dc802a4d3d8b71932",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "d954d5572f9a67de96bbaf97fc89c62c5625812890a6cc39debd9c6fb7e0dd47",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "c43d513804031e9b14f076a53c5f02b8ae121926ca7e2f33cbaad1b22cde89b8",
			},
		},
		"v6.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "7cae6188748d428311a47ab8e539fa69251615601fa9187091c59c8689b3d26a",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "5a928f99fafa8badfa81c909fab16101f9b55df6d53f8a22eef827d3879b1499",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "2b4eaf42fda7bb0bbc843f6b9602613272f346a28163a99b934178f34f3056f7",
			},
		},
		"v7.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "455b4298bb3e32d1f8be031cdc04ede834b131827dd320fb2e72c409b523da7c",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "d394d36d1877d19543005c576555ecd21f870d561d7335ac0b3b6c359ade3068",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "65dd51765052c68af16813eba697a6ec40206aacb7151a6e986caf2ca8b11ae7",
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
		"v5.1": {
			// this value should not change
			expectedHash: "fac2ba9ab18ad8d26f99b9f2d694529463b024974efdfd2a964a12df7a1e545a",
		},
		"v5.2": {
			// this value should not change
			expectedHash: "fac2ba9ab18ad8d26f99b9f2d694529463b024974efdfd2a964a12df7a1e545a",
		},
		"v6.0": {
			// this value should not change
			expectedHash: "ab4e090563643a85de3e3a54f3ee3dc7b7b5c89908820c64ab4fa8e885ed9134",
		},
		"v7.0": {
			// this value should not change
			expectedHash: "6b4f70f02dda1f464fb88f45faa5650c95f5a6cb9584ab34e470bb473af278b5",
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
