package api

import (
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

// ConvertToV20180930preview converts from an OpenShiftManagedCluster to a
// v20180930preview.OpenShiftManagedCluster.
func ConvertToV20180930preview(cs *OpenShiftManagedCluster) *v20180930preview.OpenShiftManagedCluster {
	oc := &v20180930preview.OpenShiftManagedCluster{
		ID:       cs.ID,
		Location: cs.Location,
		Name:     cs.Name,
		Tags:     cs.Tags,
		Type:     cs.Type,
	}

	if cs.Plan != nil {
		oc.Plan = &v20180930preview.ResourcePurchasePlan{
			Name:          cs.Plan.Name,
			Product:       cs.Plan.Product,
			PromotionCode: cs.Plan.PromotionCode,
			Publisher:     cs.Plan.Publisher,
		}
	}

	if cs.Properties != nil {
		oc.Properties = &v20180930preview.Properties{
			ProvisioningState: v20180930preview.ProvisioningState(cs.Properties.ProvisioningState),
		}

		oc.Properties.AuthProfile.IdentityProviders = make([]v20180930preview.IdentityProvider, len(cs.Properties.AuthProfile.IdentityProviders))
		for i, ip := range cs.Properties.AuthProfile.IdentityProviders {
			oc.Properties.AuthProfile.IdentityProviders[i].Name = ip.Name
			switch provider := ip.Provider.(type) {
			case (*AADIdentityProvider):
				oc.Properties.AuthProfile.IdentityProviders[i].Provider = &v20180930preview.AADIdentityProvider{
					ClientID: provider.ClientID,
					Secret:   provider.Secret,
					Kind:     provider.Kind,
				}
			default:
				panic("authProfile.identityProviders conversion failed")
			}
		}

		if cs.Properties.OrchestratorProfile != nil {
			oc.Properties.OpenShiftVersion = cs.Properties.OrchestratorProfile.OrchestratorVersion

			if cs.Properties.OrchestratorProfile.OpenShiftConfig != nil {
				oc.Properties.PublicHostname = cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname

				oc.Properties.RouterProfiles = make([]v20180930preview.RouterProfile, len(cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles))
				for i, rp := range cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles {
					oc.Properties.RouterProfiles[i] = v20180930preview.RouterProfile{
						Name:            rp.Name,
						PublicSubdomain: rp.PublicSubdomain,
						FQDN:            rp.FQDN,
					}
				}
			}
		}

		oc.Properties.FQDN = cs.Properties.FQDN

		if cs.Properties.ServicePrincipalProfile != nil {
			oc.Properties.ServicePrincipalProfile = v20180930preview.ServicePrincipalProfile{
				ClientID: cs.Properties.ServicePrincipalProfile.ClientID,
				Secret:   cs.Properties.ServicePrincipalProfile.Secret,
			}
		}

		// -1 because master profile moves to its own field
		oc.Properties.AgentPoolProfiles = make([]v20180930preview.AgentPoolProfile, len(cs.Properties.AgentPoolProfiles)-1)
		for i, app := range cs.Properties.AgentPoolProfiles {
			if app.Role == AgentPoolProfileRoleMaster {
				oc.Properties.MasterPoolProfile = v20180930preview.MasterPoolProfile{
					ProfileSpec: v20180930preview.ProfileSpec{
						Name:         app.Name,
						Count:        app.Count,
						VMSize:       app.VMSize,
						OSType:       v20180930preview.OSType(app.OSType),
						VnetSubnetID: app.VnetSubnetID,
					},
				}
			} else {
				oc.Properties.AgentPoolProfiles[i] = v20180930preview.AgentPoolProfile{
					ProfileSpec: v20180930preview.ProfileSpec{
						Name:         app.Name,
						Count:        app.Count,
						VMSize:       app.VMSize,
						OSType:       v20180930preview.OSType(app.OSType),
						VnetSubnetID: app.VnetSubnetID,
					},
					Role: v20180930preview.AgentPoolProfileRole(app.Role),
				}
			}
		}
	}

	return oc
}
