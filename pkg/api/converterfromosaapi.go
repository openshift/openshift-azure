package api

import (
	"github.com/openshift/openshift-azure/pkg/api/v1"
)

// ConvertV1OpenShiftManagedClusterToOpenShiftManagedCluster converts from a
// v1.OpenShiftManagedCluster to an OpenShiftManagedCluster.
func ConvertV1OpenShiftManagedClusterToOpenShiftManagedCluster(oc *v1.OpenShiftManagedCluster) *OpenShiftManagedCluster {
	cs := &OpenShiftManagedCluster{
		ID:       oc.ID,
		Location: oc.Location,
		Name:     oc.Name,
		Tags:     oc.Tags,
		Type:     oc.Type,
	}

	if oc.Plan != nil {
		cs.Plan = &ResourcePurchasePlan{
			Name:          oc.Plan.Name,
			Product:       oc.Plan.Product,
			PromotionCode: oc.Plan.PromotionCode,
			Publisher:     oc.Plan.Publisher,
		}
	}

	if oc.Properties != nil {
		cs.Properties = &Properties{
			ProvisioningState: ProvisioningState(oc.Properties.ProvisioningState),
			OrchestratorProfile: &OrchestratorProfile{
				OrchestratorVersion: oc.Properties.OpenShiftVersion,
				OpenShiftConfig: &OpenShiftConfig{
					PublicHostname: oc.Properties.PublicHostname,
				},
			},
			FQDN: oc.Properties.FQDN,
			ServicePrincipalProfile: &ServicePrincipalProfile{
				ClientID: oc.Properties.ServicePrincipalProfile.ClientID,
				Secret:   oc.Properties.ServicePrincipalProfile.Secret,
			},
		}

		cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles = make([]OpenShiftRouterProfile, len(oc.Properties.RouterProfiles))
		for i, rp := range oc.Properties.RouterProfiles {
			cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[i] = OpenShiftRouterProfile{
				Name:            rp.Name,
				PublicSubdomain: rp.PublicSubdomain,
				FQDN:            rp.FQDN,
			}
		}

		// +1 because master pool profile becomes agent pool profile
		cs.Properties.AgentPoolProfiles = make([]*AgentPoolProfile, len(oc.Properties.AgentPoolProfiles)+1)
		for i, app := range oc.Properties.AgentPoolProfiles {
			cs.Properties.AgentPoolProfiles[i] = &AgentPoolProfile{
				Name:         app.Name,
				Count:        app.Count,
				VMSize:       app.VMSize,
				OSType:       OSType(app.OSType),
				VnetSubnetID: app.VnetSubnetID,
				Role:         AgentPoolProfileRole(app.Role),
			}
		}

		cs.Properties.AgentPoolProfiles[len(oc.Properties.AgentPoolProfiles)] = &AgentPoolProfile{
			Name:         oc.Properties.MasterPoolProfile.Name,
			Count:        oc.Properties.MasterPoolProfile.Count,
			VMSize:       oc.Properties.MasterPoolProfile.VMSize,
			OSType:       OSType(oc.Properties.MasterPoolProfile.OSType),
			VnetSubnetID: oc.Properties.MasterPoolProfile.VnetSubnetID,
			Role:         AgentPoolProfileRoleMaster,
		}

		// init internal AuthProfile structure for conversion
		cs.Properties.AuthProfile = &AuthProfile{}
		if len(oc.Properties.AuthProfile.IdentityProviders) > 0 {
			cs.Properties.AuthProfile.IdentityProviders = make([]IdentityProvider, len(oc.Properties.AuthProfile.IdentityProviders))
			for i, ip := range oc.Properties.AuthProfile.IdentityProviders {
				cs.Properties.AuthProfile.IdentityProviders[i].Name = ip.Name
				switch provider := ip.Provider.(type) {
				case (*v1.AADIdentityProvider):
					cs.Properties.AuthProfile.IdentityProviders[i].Provider = &AADIdentityProvider{
						ClientID: provider.ClientID,
						Secret:   provider.Secret,
						Kind:     provider.Kind,
					}
				default:
					panic("authProfile.identityProviders conversion failed")
				}
			}
		}
	}

	return cs
}
