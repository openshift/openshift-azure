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
			OpenShiftVersion:  cs.Properties.OpenShiftVersion,
			PublicHostname:    cs.Properties.PublicHostname,
			FQDN:              cs.Properties.FQDN,
		}

		oc.Properties.RouterProfiles = make([]v20180930preview.RouterProfile, len(cs.Properties.RouterProfiles))
		for i, rp := range cs.Properties.RouterProfiles {
			oc.Properties.RouterProfiles[i] = v20180930preview.RouterProfile{
				Name:            rp.Name,
				PublicSubdomain: rp.PublicSubdomain,
				FQDN:            rp.FQDN,
			}
		}

		oc.Properties.AgentPoolProfiles = make([]v20180930preview.AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles)-1)
		for _, app := range cs.Properties.AgentPoolProfiles {
			if app.Role == AgentPoolProfileRoleMaster {
				oc.Properties.MasterPoolProfile = &v20180930preview.MasterPoolProfile{
					Name:         app.Name,
					Count:        app.Count,
					VMSize:       app.VMSize,
					OSType:       v20180930preview.OSType(app.OSType),
					VnetSubnetID: app.VnetSubnetID,
				}

			} else {
				oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, v20180930preview.AgentPoolProfile{
					Name:         app.Name,
					Count:        app.Count,
					VMSize:       app.VMSize,
					OSType:       v20180930preview.OSType(app.OSType),
					VnetSubnetID: app.VnetSubnetID,
					Role:         v20180930preview.AgentPoolProfileRole(app.Role),
				})
			}
		}

		if cs.Properties.AuthProfile != nil {
			oc.Properties.AuthProfile = &v20180930preview.AuthProfile{}

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
		}

		if cs.Properties.ServicePrincipalProfile != nil {
			oc.Properties.ServicePrincipalProfile = &v20180930preview.ServicePrincipalProfile{
				ClientID: cs.Properties.ServicePrincipalProfile.ClientID,
				Secret:   cs.Properties.ServicePrincipalProfile.Secret,
			}
		}
	}

	return oc
}
