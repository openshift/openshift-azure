package v20180930preview

import (
	"github.com/openshift/openshift-azure/pkg/api"
)

// ConvertToV20180930preview converts from an OpenShiftManagedCluster to a
// OpenShiftManagedCluster.
func ConvertToV20180930preview(cs *api.OpenShiftManagedCluster) *OpenShiftManagedCluster {
	oc := &OpenShiftManagedCluster{
		ID:       &cs.ID,
		Location: &cs.Location,
		Name:     &cs.Name,
		Type:     &cs.Type,
	}
	oc.Tags = make(map[string]*string, len(cs.Tags))
	for k := range cs.Tags {
		v := cs.Tags[k]
		oc.Tags[k] = &v
	}
	if cs.Plan != nil {
		oc.Plan = &ResourcePurchasePlan{
			Name:          cs.Plan.Name,
			Product:       cs.Plan.Product,
			PromotionCode: cs.Plan.PromotionCode,
			Publisher:     cs.Plan.Publisher,
		}
	}

	provisioningState := ProvisioningState(cs.Properties.ProvisioningState)
	oc.Properties = &Properties{
		ProvisioningState: &provisioningState,
		OpenShiftVersion:  &cs.Properties.OpenShiftVersion,
		ClusterVersion:    &cs.Properties.ClusterVersion,
		PublicHostname:    &cs.Properties.PublicHostname,
		FQDN:              &cs.Properties.FQDN,
	}

	oc.Properties.NetworkProfile = &NetworkProfile{
		VnetID:     &cs.Properties.NetworkProfile.VnetID,
		VnetCIDR:   &cs.Properties.NetworkProfile.VnetCIDR,
		PeerVnetID: cs.Properties.NetworkProfile.PeerVnetID,
	}

	oc.Properties.RouterProfiles = make([]RouterProfile, len(cs.Properties.RouterProfiles))
	for i := range cs.Properties.RouterProfiles {
		rp := cs.Properties.RouterProfiles[i]
		oc.Properties.RouterProfiles[i] = RouterProfile{
			Name:            &rp.Name,
			PublicSubdomain: &rp.PublicSubdomain,
			FQDN:            &rp.FQDN,
		}
	}

	oc.Properties.AgentPoolProfiles = make([]AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles))
	for i := range cs.Properties.AgentPoolProfiles {
		app := cs.Properties.AgentPoolProfiles[i]
		vmSize := VMSize(app.VMSize)

		if app.Role == api.AgentPoolProfileRoleMaster {
			oc.Properties.MasterPoolProfile = &MasterPoolProfile{
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
			}
		} else {
			osType := OSType(app.OSType)
			role := AgentPoolProfileRole(app.Role)

			oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, AgentPoolProfile{
				Name:       &app.Name,
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
				OSType:     &osType,
				Role:       &role,
			})
		}
	}

	oc.Properties.AuthProfile = &AuthProfile{}
	oc.Properties.AuthProfile.IdentityProviders = make([]IdentityProvider, len(cs.Properties.AuthProfile.IdentityProviders))
	for i := range cs.Properties.AuthProfile.IdentityProviders {
		ip := cs.Properties.AuthProfile.IdentityProviders[i]
		oc.Properties.AuthProfile.IdentityProviders[i].Name = &ip.Name
		switch provider := ip.Provider.(type) {
		case (*api.AADIdentityProvider):
			oc.Properties.AuthProfile.IdentityProviders[i].Provider = &AADIdentityProvider{
				Kind:                 &provider.Kind,
				ClientID:             &provider.ClientID,
				Secret:               &provider.Secret,
				TenantID:             &provider.TenantID,
				CustomerAdminGroupID: provider.CustomerAdminGroupID,
			}

		default:
			panic("authProfile.identityProviders conversion failed")
		}
	}

	return oc
}
