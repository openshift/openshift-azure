package plugin

import (
	"context"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/fixtures"
)

func TestMerge(t *testing.T) {
	p := NewPlugin(logrus.NewEntry(logrus.New()))
	newCluster := fixtures.NewTestOpenShiftCluster()
	oldCluster := fixtures.NewTestOpenShiftCluster()

	newCluster.Config = nil
	newCluster.Properties.AgentPoolProfiles = nil
	newCluster.Properties.OrchestratorProfile = nil
	newCluster.Properties.FQDN = ""

	// make old cluster go through plugin first
	testPluginRun(p, oldCluster, nil, t)

	// should fix all of the items removed above and we should
	// be able to run through the entire plugin process.
	p.MergeConfig(context.Background(), newCluster, oldCluster)

	if newCluster.Config == nil {
		t.Errorf("new cluster config should be merged")
	}
	if len(newCluster.Properties.AgentPoolProfiles) == 0 {
		t.Errorf("new cluster agent pool profiels should be merged")
	}
	if newCluster.Properties.OrchestratorProfile == nil {
		t.Errorf("new cluster orchestrator profile should be merged")
	}
	if newCluster.Properties.FQDN == "" {
		t.Errorf("new cluster fqdn should be merged")
	}

	testPluginRun(p, newCluster, oldCluster, t)
}

func testPluginRun(p api.Plugin, newCluster *api.OpenShiftManagedCluster, oldCluster *api.OpenShiftManagedCluster, t *testing.T) {
	if errs := p.Validate(context.Background(), newCluster, oldCluster, false); len(errs) != 0 {
		t.Fatalf("error validating: %s", spew.Sdump(errs))
	}

	if err := p.GenerateConfig(context.Background(), newCluster); err != nil {
		t.Fatalf("error generating config for arm generate test: %s", spew.Sdump(err))
	}

	bytes, err := p.GenerateARM(context.Background(), newCluster)
	if err != nil {
		t.Fatalf("error generating arm: %s", spew.Sdump(err))
	}
	if len(bytes) == 0 {
		t.Errorf("no arm was generated")
	}
}
