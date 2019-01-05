package api

import (
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

// ConvertToV20180930preview converts from an OpenShiftManagedCluster to a
// v20180930preview.OpenShiftManagedCluster.
func ConvertToV20180930preview(cs *OpenShiftManagedCluster) *v20180930preview.OpenShiftManagedCluster {
	oc := &v20180930preview.OpenShiftManagedCluster{
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
		oc.Plan = &v20180930preview.ResourcePurchasePlan{
			Name:          cs.Plan.Name,
			Product:       cs.Plan.Product,
			PromotionCode: cs.Plan.PromotionCode,
			Publisher:     cs.Plan.Publisher,
		}
	}

	provisioningState := v20180930preview.ProvisioningState(cs.Properties.ProvisioningState)
	oc.Properties = &v20180930preview.Properties{
		ProvisioningState: &provisioningState,
		OpenShiftVersion:  &cs.Properties.OpenShiftVersion,
		PublicHostname:    &cs.Properties.PublicHostname,
		FQDN:              &cs.Properties.FQDN,
	}

	oc.Properties.NetworkProfile = &v20180930preview.NetworkProfile{
		VnetCIDR:   &cs.Properties.NetworkProfile.VnetCIDR,
		PeerVnetID: &cs.Properties.NetworkProfile.PeerVnetID,
	}

	oc.Properties.RouterProfiles = make([]v20180930preview.RouterProfile, len(cs.Properties.RouterProfiles))
	for i := range cs.Properties.RouterProfiles {
		rp := cs.Properties.RouterProfiles[i]
		oc.Properties.RouterProfiles[i] = v20180930preview.RouterProfile{
			Name:            &rp.Name,
			PublicSubdomain: &rp.PublicSubdomain,
			FQDN:            &rp.FQDN,
		}
	}

	oc.Properties.AgentPoolProfiles = make([]v20180930preview.AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles))
	for i := range cs.Properties.AgentPoolProfiles {
		app := cs.Properties.AgentPoolProfiles[i]
		vmSize := v20180930preview.VMSize(app.VMSize)

		if app.Role == AgentPoolProfileRoleMaster {
			oc.Properties.MasterPoolProfile = &v20180930preview.MasterPoolProfile{
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
			}
		} else {
			osType := v20180930preview.OSType(app.OSType)
			role := v20180930preview.AgentPoolProfileRole(app.Role)

			oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, v20180930preview.AgentPoolProfile{
				Name:       &app.Name,
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
				OSType:     &osType,
				Role:       &role,
			})
		}
	}

	oc.Properties.AuthProfile = &v20180930preview.AuthProfile{}
	oc.Properties.AuthProfile.IdentityProviders = make([]v20180930preview.IdentityProvider, len(cs.Properties.AuthProfile.IdentityProviders))
	for i := range cs.Properties.AuthProfile.IdentityProviders {
		ip := cs.Properties.AuthProfile.IdentityProviders[i]
		oc.Properties.AuthProfile.IdentityProviders[i].Name = &ip.Name
		switch provider := ip.Provider.(type) {
		case (*AADIdentityProvider):
			oc.Properties.AuthProfile.IdentityProviders[i].Provider = &v20180930preview.AADIdentityProvider{
				Kind:     &provider.Kind,
				ClientID: &provider.ClientID,
				Secret:   &provider.Secret,
				TenantID: &provider.TenantID,
			}

		default:
			panic("authProfile.identityProviders conversion failed")
		}
	}

	return oc
}
