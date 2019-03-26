package api

import (
	"github.com/Azure/go-autorest/autorest/to"
)

// GetInternalMockCluster returns mock object of the
// internal API model
func GetInternalMockCluster() *OpenShiftManagedCluster {
	// this is the expected internal equivalent to
	// v20190430ManagedCluster()

	return &OpenShiftManagedCluster{
		ID:       "ID",
		Location: "Location",
		Name:     "Name",
		Plan: &ResourcePurchasePlan{
			Name:          to.StringPtr("Plan.Name"),
			Product:       to.StringPtr("Plan.Product"),
			PromotionCode: to.StringPtr("Plan.PromotionCode"),
			Publisher:     to.StringPtr("Plan.Publisher"),
		},
		Tags: map[string]string{
			"Tags.key": "Tags.val",
		},
		Type: "Type",
		Properties: Properties{
			ProvisioningState: "Properties.ProvisioningState",
			OpenShiftVersion:  "Properties.OpenShiftVersion",
			ClusterVersion:    "Properties.ClusterVersion",
			PublicHostname:    "Properties.PublicHostname",
			RouterProfiles: []RouterProfile{
				{
					Name:            "Properties.RouterProfiles[0].Name",
					PublicSubdomain: "Properties.RouterProfiles[0].PublicSubdomain",
					FQDN:            "Properties.RouterProfiles[0].FQDN",
				},
			},
			FQDN: "Properties.FQDN",
			AuthProfile: AuthProfile{
				IdentityProviders: []IdentityProvider{
					{
						Name: "Properties.AuthProfile.IdentityProviders[0].Name",
						Provider: &AADIdentityProvider{
							Kind:                 "AADIdentityProvider",
							ClientID:             "Properties.AuthProfile.IdentityProviders[0].Provider.ClientID",
							Secret:               "Properties.AuthProfile.IdentityProviders[0].Provider.Secret",
							TenantID:             "Properties.AuthProfile.IdentityProviders[0].Provider.TenantID",
							CustomerAdminGroupID: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Provider.CustomerAdminGroupID"),
						},
					},
				},
			},
			NetworkProfile: NetworkProfile{
				VnetID:     "Properties.NetworkProfile.VnetID",
				VnetCIDR:   "Properties.NetworkProfile.VnetCIDR",
				PeerVnetID: to.StringPtr("Properties.NetworkProfile.PeerVnetID"),
			},
			AgentPoolProfiles: []AgentPoolProfile{
				{
					Name:       string(AgentPoolProfileRoleMaster),
					Count:      1,
					VMSize:     "Properties.MasterPoolProfile.VMSize",
					SubnetCIDR: "Properties.MasterPoolProfile.SubnetCIDR",
					OSType:     OSTypeLinux,
					Role:       AgentPoolProfileRoleMaster,
				},
				{
					Name:       "Properties.AgentPoolProfiles[0].Name",
					Count:      1,
					VMSize:     "Properties.AgentPoolProfiles[0].VMSize",
					SubnetCIDR: "Properties.AgentPoolProfiles[0].SubnetCIDR",
					OSType:     "Properties.AgentPoolProfiles[0].OSType",
					Role:       "Properties.AgentPoolProfiles[0].Role",
				},
			},
		},
	}
}
