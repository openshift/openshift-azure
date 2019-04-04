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
		"v3.2": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// jminter: e436... value verified manually against v3.2
				// this value should not change
				expectedHash: "e43619d6476b958470ec8e95f83c1023f15c63bb7d896adc2246cc1d9da68d0e",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// jminter: d438... value verified manually against v3.2
				// this value should not change
				expectedHash: "d438d7076d6a3495d0663323dd81600b389f992499750df0a859d208d9fa42f7",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// jminter: fc64... value verified manually against v3.2
				// this value should not change
				expectedHash: "fc64527a7a3fc02568083f200de97c9fcb22aea448e57ae46e58f1a76ba1f6f6",
			},
		},
		"v4.0": {
			{
				role: api.AgentPoolProfileRoleMaster,
				// this value should not change
				expectedHash: "49de607e3b50aff9f2fa1a64e7a6173ab2c13969dcdc9036623ddc0666ca99af",
			},
			{
				role: api.AgentPoolProfileRoleInfra,
				// this value should not change
				expectedHash: "a8236fac988b1e73d7685bb70f5c7ec2c17a0bdd41f60e4d92f216c9f4961f50",
			},
			{
				role: api.AgentPoolProfileRoleCompute,
				// this value should not change
				expectedHash: "90abb10d7670b484d65bf107102b44164284db52ee7038b1753c7f15000c0abc",
			},
		},
		"v5.0": {
			{
				role:         api.AgentPoolProfileRoleMaster,
				expectedHash: "49de607e3b50aff9f2fa1a64e7a6173ab2c13969dcdc9036623ddc0666ca99af",
			},
			{
				role:         api.AgentPoolProfileRoleInfra,
				expectedHash: "a8236fac988b1e73d7685bb70f5c7ec2c17a0bdd41f60e4d92f216c9f4961f50",
			},
			{
				role:         api.AgentPoolProfileRoleCompute,
				expectedHash: "90abb10d7670b484d65bf107102b44164284db52ee7038b1753c7f15000c0abc",
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
		if _, found := tests[version]; !found {
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

			hasher := hasher{
				startupFactory: startup.New,
				arm:            arm,
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
		"v3.2": {
			// jminter: 302c... value verified manually against v3.2
			// this value should not change
			expectedHash: "302c52d35602ad1c4a6078953abccc36ecd92ba8256b6e118837712a69d7f028",
		},
		"v4.0": {
			// this value should not change
			expectedHash: "94c9eefd9c847c49834925c5f7dd87b21765fe8a3952f8190f9b352a8a3cba37",
		},
		"v5.0": {
			expectedHash: "4baa0e9115ff4edb20eef6d8267d635cd016ce4e8babdaabe0a56ea5c7364a0a",
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
		if _, found := tests[version]; !found {
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
