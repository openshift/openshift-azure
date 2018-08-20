package fixtures

import "github.com/openshift/openshift-azure/pkg/api"

// NewTestOpenShiftCluster is a test cluster definition that can be use in unit testing plugin methods.
func NewTestOpenShiftCluster() *api.OpenShiftManagedCluster {
	return &api.OpenShiftManagedCluster{
		ID:       "test",
		Location: "eastus",
		Name:     "openshfit",
		Config:   &api.Config{},
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
