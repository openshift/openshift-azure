package plugin

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestMerge(t *testing.T) {
	p := NewPlugin(logrus.NewEntry(logrus.New()))
	newCluster := testOpenShiftCluster()
	oldCluster := testOpenShiftCluster()

	newCluster.Config = nil
	newCluster.Properties.AgentPoolProfiles = nil
	newCluster.Properties.OrchestratorProfile = nil
	newCluster.Properties.FQDN = ""

	// make old cluster go through plugin first
	testPluginRun(p, oldCluster, nil, t)

	// should fix all of the items removed above and we should
	// be able to run through the entire plugin process.
	p.MergeConfig(newCluster, oldCluster)

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
	if errs := p.Validate(newCluster, oldCluster, false); len(errs) != 0 {
		t.Fatalf("error validating: %s", spew.Sdump(errs))
	}

	if err := p.GenerateConfig(newCluster); err != nil {
		t.Fatalf("error generating config for arm generate test: %s", spew.Sdump(err))
	}

	bytes, err := p.GenerateARM(newCluster)
	if err != nil {
		t.Fatalf("error generating arm: %s", spew.Sdump(err))
	}
	if len(bytes) == 0 {
		t.Errorf("no arm was generated")
	}
}

func testOpenShiftCluster() *api.OpenShiftManagedCluster {
	return &api.OpenShiftManagedCluster{
		ID:       "test",
		Location: "eastus",
		Name:     "openshfit",
		Properties: &api.Properties{
			FQDN: "www.example.com",
			OrchestratorProfile: &api.OrchestratorProfile{
				OrchestratorVersion: "v3.10",
				OpenShiftConfig: &api.OpenShiftConfig{
					PublicHostname: "openshift.test.example.com",
					RouterProfiles: []api.OpenShiftRouterProfile{
						{
							Name:            "default",
							PublicSubdomain: "test.example.com",
						},
					},
				},
			},
			AgentPoolProfiles: []*api.AgentPoolProfile{
				{
					Name:   "master",
					Role:   api.AgentPoolProfileRoleMaster,
					Count:  3,
					VMSize: "Standard_D2s_v3",
					OSType: "Linux",
				},
				{
					Name:   "infra",
					Role:   api.AgentPoolProfileRoleInfra,
					Count:  1,
					VMSize: "Standard_D2s_v3",
					OSType: "Linux",
				},
				{
					Name:   "compute",
					Role:   api.AgentPoolProfileRoleCompute,
					Count:  1,
					VMSize: "Standard_D2s_v3",
					OSType: "Linux",
				},
			},
			ServicePrincipalProfile: &api.ServicePrincipalProfile{
				ClientID: "client_id",
				Secret:   "client_secrett",
			},
			AzProfile: &api.AzProfile{
				TenantID:       "tenant",
				SubscriptionID: "sub",
				ResourceGroup:  "rg",
			},
		},
	}
}
