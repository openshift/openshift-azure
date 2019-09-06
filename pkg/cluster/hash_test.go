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
		"v7.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "9758f2d02a551c3c157636994c93d5c07678ee4a28341e81356b0cce7d0a8fc1",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "6eeed214cb09c42ecc1a665fed139d53c42884c9a536601a02b59bf5e4b97422",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "9a5c0476b5e152736202d87ee25903830e17a757534e9de24060e36e3080fcb3",
			},
		},
		"v8.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "c1e17c7ae4ba5c722a8f14e99f7d0b5bc5c63ded09f1ef4ed4a59a96ab2d3ea8",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "b5ec0820567b6ed6dd5e798d5fe2f20aa345d77b17f80f9d1bafce37893f3bcf",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "5945fa8733ef655278a643092f4774e58bd0e2572607bee6ebd7927e40dce868",
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
		"v7.0": {
			// this value should not change
			expectedHash: "40f9a51d8328ad65e4cdda62e460b53a6b2a5b71908f43220f4a17685c49d562",
		},
		"v8.0": {
			// this value should not change
			expectedHash: "74ca5980be48bc8efd9bd0b7d1929bf3c28ae3d60e3be666e6df41aa74fa14d4",
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
		switch cs.Config.PluginVersion {
		case "v7.0":
			cs.Config.Images.Console = ""
		default:
			// needed by derived.OpenShiftClientVersion()
			cs.Config.Images.Console = "foo:v1.2.3"
		}

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
