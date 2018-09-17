package api

import (
	"github.com/Azure/go-autorest/autorest/to"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

// ConvertFromV20180930preview converts from a
// v20180930preview.OpenShiftManagedCluster to an OpenShiftManagedCluster.
func ConvertFromV20180930preview(oc *v20180930preview.OpenShiftManagedCluster) *OpenShiftManagedCluster {
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
			OpenShiftVersion:  oc.Properties.OpenShiftVersion,
			PublicHostname:    oc.Properties.PublicHostname,
			FQDN:              oc.Properties.FQDN,
		}

		cs.Properties.RouterProfiles = make([]RouterProfile, len(oc.Properties.RouterProfiles))
		for i, rp := range oc.Properties.RouterProfiles {
			cs.Properties.RouterProfiles[i] = RouterProfile{
				Name:            rp.Name,
				PublicSubdomain: rp.PublicSubdomain,
				FQDN:            rp.FQDN,
			}
		}

		cs.Properties.AgentPoolProfiles = make([]AgentPoolProfile, 0, len(oc.Properties.AgentPoolProfiles)+1)
		for _, app := range oc.Properties.AgentPoolProfiles {
			newApp := AgentPoolProfile{
				Name:         app.Name,
				VMSize:       VMSize(app.VMSize),
				OSType:       OSType(app.OSType),
				VnetSubnetID: app.VnetSubnetID,
				Role:         AgentPoolProfileRole(app.Role),
			}
			if app.Count != nil {
				newApp.Count = to.IntPtr(*app.Count)
			}
			cs.Properties.AgentPoolProfiles = append(cs.Properties.AgentPoolProfiles, newApp)
		}

		if oc.Properties.MasterPoolProfile != nil {
			newApp := AgentPoolProfile{
				Name:         string(AgentPoolProfileRoleMaster),
				VMSize:       VMSize(oc.Properties.MasterPoolProfile.VMSize),
				OSType:       OSTypeLinux,
				VnetSubnetID: oc.Properties.MasterPoolProfile.VnetSubnetID,
				Role:         AgentPoolProfileRoleMaster,
			}
			if oc.Properties.MasterPoolProfile.Count != nil {
				newApp.Count = to.IntPtr(*oc.Properties.MasterPoolProfile.Count)
			}
			cs.Properties.AgentPoolProfiles = append(cs.Properties.AgentPoolProfiles, newApp)
		}

		if oc.Properties.AuthProfile != nil {
			cs.Properties.AuthProfile = &AuthProfile{}

			cs.Properties.AuthProfile.IdentityProviders = make([]IdentityProvider, len(oc.Properties.AuthProfile.IdentityProviders))
			for i, ip := range oc.Properties.AuthProfile.IdentityProviders {
				cs.Properties.AuthProfile.IdentityProviders[i].Name = ip.Name
				switch provider := ip.Provider.(type) {
				case (*v20180930preview.AADIdentityProvider):
					cs.Properties.AuthProfile.IdentityProviders[i].Provider = &AADIdentityProvider{
						Kind:     provider.Kind,
						ClientID: provider.ClientID,
						Secret:   provider.Secret,
						TenantID: provider.TenantID,
					}

				default:
					panic("authProfile.identityProviders conversion failed")
				}
			}
		}
	}

	return cs
}
