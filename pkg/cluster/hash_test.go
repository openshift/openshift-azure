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
				expectedHash: "51df71b5d8b4586dbdb5736f54f65d201b31ee5d7facd0378a4c59507aaa2e61",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "84c7e8f1ec270a1685a08542746a6c000306217562d9d475baf52b22eb05e490",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "78dd1e1c4d1c80cb1aa20d448daebe9cc378c50c8c65c300814eb2a944046362",
			},
		},
		"v7.1": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "d487b62136a4f5e0d603ec2c0e0074366850fc16b04f1181d7dab9a53707c916",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "e36358c9f1a05cc7d9cd7dd58d1a5ef0145b15c710979817dcfe1c4edd911878",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "b02af291c3fe22fb1e289493959a17d42d5bcd69af1166d1dbb24bf80c69da93",
			},
		},
		"v9.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "5a95b011d1d39d4a4b98a5617ae18fa0a90dacc884d251280ee9785c89668b56",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "9271c1678f629b0dde716fb696726e8c3eb3431fee7102e3257781c2d05cb254",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "695fcd5966728dbcb47bf553e1db0ea921934f67414cd912cb627f6822021bd7",
			},
		},
		"v10.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "e84af7bbabb6acb06fe41a9c0ba171100fa93f280872c36e854472200c23e57b",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "b2fdff00935d4ff94ecc9448864e3753b1d911b5d575eaa783041f9cfd298032",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "2b5408737a68306367c7f34e3c6aeb21f1e6ebaf6f7a703b092fce948700d968",
			},
		},
		"v10.1": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "e84af7bbabb6acb06fe41a9c0ba171100fa93f280872c36e854472200c23e57b",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "b2fdff00935d4ff94ecc9448864e3753b1d911b5d575eaa783041f9cfd298032",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "2b5408737a68306367c7f34e3c6aeb21f1e6ebaf6f7a703b092fce948700d968",
			},
		},
		"v10.2": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "e84af7bbabb6acb06fe41a9c0ba171100fa93f280872c36e854472200c23e57b",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "b2fdff00935d4ff94ecc9448864e3753b1d911b5d575eaa783041f9cfd298032",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "2b5408737a68306367c7f34e3c6aeb21f1e6ebaf6f7a703b092fce948700d968",
			},
		},
		"v12.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "48c48f1eb85a1810052bf2cfcf5f8c9a420b3e3339141c1f81245241b50b28b9",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "99f4eb33af2723947783a6db1c3b9fb9d36d1989b4f4f658d776efec072779f1",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "959b10157d2864150339914d0e5ddee5dc7d7e1e6d2318f0ef9e5325cc6b09df",
			},
		},
		"v12.1": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "48c48f1eb85a1810052bf2cfcf5f8c9a420b3e3339141c1f81245241b50b28b9",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "99f4eb33af2723947783a6db1c3b9fb9d36d1989b4f4f658d776efec072779f1",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "959b10157d2864150339914d0e5ddee5dc7d7e1e6d2318f0ef9e5325cc6b09df",
			},
		},
		"v12.2": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "48c48f1eb85a1810052bf2cfcf5f8c9a420b3e3339141c1f81245241b50b28b9",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "99f4eb33af2723947783a6db1c3b9fb9d36d1989b4f4f658d776efec072779f1",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "959b10157d2864150339914d0e5ddee5dc7d7e1e6d2318f0ef9e5325cc6b09df",
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
		switch version {
		case "v7.0", "v9.0", "v7.1":
			cs.Config.Images.ImagePullSecret = []byte{}
			cs.Config.Images.GenevaImagePullSecret = []byte{}
		default:
			cs.Config.Images.ImagePullSecret = populate.DummyImagePullSecret("registry.redhat.io")
			cs.Config.Images.GenevaImagePullSecret = populate.DummyImagePullSecret("osarpint.azurecr.io")
		}
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
		"v7.1": {
			// this value should not change
			expectedHash: "13606ac122bf615190ff88d5c358709aaba9228c9e8cab031c058184bd016444",
		},
		"v9.0": {
			// this value should not change
			expectedHash: "6a8bc3cc2340bea2a7a1e9d63da4390a8134eedd517774a6f7b83c2c6941ee47",
		},
		"v10.0": {
			// this value should not change
			expectedHash: "76a965e10801d086a00f200c4a004ffaaa2b88c25d8451ed6729b85e745a863b",
		},
		"v10.1": {
			// this value should not change
			expectedHash: "76a965e10801d086a00f200c4a004ffaaa2b88c25d8451ed6729b85e745a863b",
		},
		"v10.2": {
			// this value should not change
			expectedHash: "76a965e10801d086a00f200c4a004ffaaa2b88c25d8451ed6729b85e745a863b",
		},
		"v12.0": {
			// this value should not change
			expectedHash: "e6e1637db32ca54384e7b975f9ce652cdee06d3e494ec28a8b663e4f70e84af8",
		},
		"v12.1": {
			// this value should not change
			expectedHash: "e6e1637db32ca54384e7b975f9ce652cdee06d3e494ec28a8b663e4f70e84af8",
		},
		"v12.2": {
			// this value should not change
			expectedHash: "e6e1637db32ca54384e7b975f9ce652cdee06d3e494ec28a8b663e4f70e84af8",
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
		switch cs.Config.PluginVersion {
		case "v7.0", "v7.1":
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
