package plugin

import (
	"context"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
)

func TestMerge(t *testing.T) {
	p := NewPlugin(logrus.NewEntry(logrus.New()), "sync:latest")
	newCluster := fixtures.NewTestOpenShiftCluster()
	oldCluster := fixtures.NewTestOpenShiftCluster()

	newCluster.Config = nil
	newCluster.Properties.AgentPoolProfiles = nil
	newCluster.Properties.PublicHostname = ""
	newCluster.Properties.RouterProfiles = nil
	newCluster.Properties.ServicePrincipalProfile = nil
	newCluster.Properties.AzProfile = nil
	newCluster.Properties.AuthProfile = nil
	newCluster.Properties.FQDN = ""

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
	if newCluster.Properties.PublicHostname == "" {
		t.Errorf("new cluster public hostname should be merged")
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
