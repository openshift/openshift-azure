package plugin

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
)

func TestMerge(t *testing.T) {
	var config = api.PluginConfig{SyncImage: "sync:latest",
		AcceptLanguages: []string{"en-us"}}
	p := NewPlugin(logrus.NewEntry(logrus.New()), config)
	newCluster := fixtures.NewTestOpenShiftCluster()
	oldCluster := fixtures.NewTestOpenShiftCluster()

	newCluster.Config = nil
	newCluster.Properties.AgentPoolProfiles = nil
	newCluster.Properties.RouterProfiles = nil
	newCluster.Properties.ServicePrincipalProfile = nil
	newCluster.Properties.AzProfile = nil
	newCluster.Properties.AuthProfile = nil
	newCluster.Properties.FQDN = ""

	testPluginWithEnv(t)

	// make old cluster go through plugin first
	armTemplate := testPluginRun(p, oldCluster, nil, t)
	if !strings.Contains(string(armTemplate), "\"type\": \"Microsoft.Network/networkSecurityGroups\"") {
		t.Fatalf("networkSecurityGroups should be applied during cluster creation")
	}

	// should fix all of the items removed above and we should
	// be able to run through the entire plugin process.
	p.MergeConfig(context.Background(), newCluster, oldCluster)

	if newCluster.Config == nil {
		t.Errorf("new cluster config should be merged")
	}
	if len(newCluster.Properties.AgentPoolProfiles) == 0 {
		t.Errorf("new cluster agent pool profiles should be merged")
	}
	if len(newCluster.Properties.RouterProfiles) == 0 {
		t.Errorf("new cluster router profiles should be merged")
	}
	if newCluster.Properties.ServicePrincipalProfile == nil {
		t.Errorf("new cluster service principal profile should be merged")
	}
	if newCluster.Properties.AzProfile == nil {
		t.Errorf("new cluster az profile should be merged")
	}
	if newCluster.Properties.AuthProfile == nil {
		t.Errorf("new cluster auth profile should be merged")
	}
	if newCluster.Properties.FQDN == "" {
		t.Errorf("new cluster fqdn should be merged")
	}

	armTemplate = testPluginRun(p, newCluster, oldCluster, t)
	if strings.Contains(string(armTemplate), "\"type\": \"Microsoft.Network/networkSecurityGroups\"") {
		t.Fatalf("networkSecurityGroups should not be applied during cluster upgrade")
	}
}

func testPluginRun(p api.Plugin, newCluster *api.OpenShiftManagedCluster, oldCluster *api.OpenShiftManagedCluster, t *testing.T) (armTemplate []byte) {
	if errs := p.Validate(context.Background(), newCluster, oldCluster, false); len(errs) != 0 {
		t.Fatalf("error validating: %s", spew.Sdump(errs))
	}

	if err := p.GenerateConfig(context.Background(), newCluster); err != nil {
		t.Fatalf("error generating config for arm generate test: %s", spew.Sdump(err))
	}

	bytes, err := p.GenerateARM(context.Background(), newCluster, oldCluster != nil)
	if err != nil {
		t.Fatalf("error generating arm: %s", spew.Sdump(err))
	}
	if len(bytes) == 0 {
		t.Errorf("no arm was generated")
	}
	return bytes
}

// Putting this in here to avoid polluting the environment in other tests
func testPluginWithEnv(t *testing.T) {
	required := GetRequiredConfigEnvVars()
	// environment placeholders
	placeholders := map[string]string{}
	// blank env vars, only store those env vars that already exist (and their values)
	for i := range required {
		value, found := os.LookupEnv(required[i])
		if found {
			placeholders[required[i]] = value
		}
		os.Unsetenv(required[i])
	}

	pluginConfig, err := NewPluginConfigFromEnv()

	// creating a plugin config from an incomplete env should fail!
	if err == nil {
		t.Errorf("a plugin config should require a complete set of env vars: %s", spew.Sdump(pluginConfig))
	}

	// creating a plugin config with a complete env should succeed
	for i := range required {
		os.Setenv(required[i], "non-empty")
	}
	pluginConfig, err = NewPluginConfigFromEnv()
	if err != nil {
		t.Errorf("plugin config from env failed: %s", spew.Sdump(pluginConfig))
	}

	// restore environment variables as they were
	for i := range required {
		os.Unsetenv(required[i])
	}
	for k, v := range placeholders {
		os.Setenv(k, v)
	}
}
