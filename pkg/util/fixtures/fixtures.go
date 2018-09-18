package fixtures

import "github.com/openshift/openshift-azure/pkg/api"

// NewTestOpenShiftCluster is a test cluster definition that can be use in unit testing plugin methods.
func NewTestOpenShiftCluster() *api.OpenShiftManagedCluster {
	return &api.OpenShiftManagedCluster{
		ID:       "test",
		Location: "eastus",
		Name:     "openshift",
		Config:   &api.Config{},
		Properties: &api.Properties{
			FQDN:             "example.eastus.cloudapp.azure.com",
			OpenShiftVersion: "v3.10",
			RouterProfiles: []api.RouterProfile{
				{
					Name:            "default",
					FQDN:            "router-fqdn.eastus.cloudapp.azure.com",
					PublicSubdomain: "test.example.com",
				},
			},
			AuthProfile: &api.AuthProfile{
				IdentityProviders: []api.IdentityProvider{
					{
						Name: "Azure AD",
						Provider: &api.AADIdentityProvider{
							Kind:     "AADIdentityProvider",
							ClientID: "properties.authProfile.identityProviders.0.provider.clientId",
							Secret:   "properties.authProfile.identityProviders.0.provider.secret",
							TenantID: "properties.authProfile.identityProviders.0.provider.tenantId",
						},
					},
				},
			},
			AgentPoolProfiles: []api.AgentPoolProfile{
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
					Count:  2,
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
