package api

import (
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

func nilIfv20180930ProvisioningStateEmpty(s *v20180930preview.ProvisioningState) *v20180930preview.ProvisioningState {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

func nilIfv20180930AgentPoolProfileRoleEmpty(s *v20180930preview.AgentPoolProfileRole) *v20180930preview.AgentPoolProfileRole {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

func nilIfv20180930VMSizeEmpty(s *v20180930preview.VMSize) *v20180930preview.VMSize {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

func nilIfv20180930OSTypeEmpty(s *v20180930preview.OSType) *v20180930preview.OSType {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

// ConvertToV20180930preview converts from an OpenShiftManagedCluster to a
// v20180930preview.OpenShiftManagedCluster.
func ConvertToV20180930preview(cs *OpenShiftManagedCluster) *v20180930preview.OpenShiftManagedCluster {
	oc := &v20180930preview.OpenShiftManagedCluster{
		ID:       nilIfStringEmpty(&cs.ID),
		Location: nilIfStringEmpty(&cs.Location),
		Name:     nilIfStringEmpty(&cs.Name),
		Type:     nilIfStringEmpty(&cs.Type),
	}
	oc.Tags = make(map[string]*string, len(cs.Tags))
	for k := range cs.Tags {
		v := cs.Tags[k]
		oc.Tags[k] = nilIfStringEmpty(&v)
	}

	oc.Plan = &v20180930preview.ResourcePurchasePlan{
		Name:          nilIfStringEmpty(&cs.Plan.Name),
		Product:       nilIfStringEmpty(&cs.Plan.Product),
		PromotionCode: nilIfStringEmpty(&cs.Plan.PromotionCode),
		Publisher:     nilIfStringEmpty(&cs.Plan.Publisher),
	}

	provisioningState := v20180930preview.ProvisioningState(cs.Properties.ProvisioningState)
	oc.Properties = &v20180930preview.Properties{
		ProvisioningState: nilIfv20180930ProvisioningStateEmpty(&provisioningState),
		OpenShiftVersion:  nilIfStringEmpty(&cs.Properties.OpenShiftVersion),
		PublicHostname:    nilIfStringEmpty(&cs.Properties.PublicHostname),
		FQDN:              nilIfStringEmpty(&cs.Properties.FQDN),
	}

	oc.Properties.NetworkProfile = &v20180930preview.NetworkProfile{
		VnetCIDR:   nilIfStringEmpty(&cs.Properties.NetworkProfile.VnetCIDR),
		PeerVnetID: nilIfStringEmpty(&cs.Properties.NetworkProfile.PeerVnetID),
	}

	oc.Properties.RouterProfiles = make([]v20180930preview.RouterProfile, len(cs.Properties.RouterProfiles))
	for i := range cs.Properties.RouterProfiles {
		rp := cs.Properties.RouterProfiles[i]
		oc.Properties.RouterProfiles[i] = v20180930preview.RouterProfile{
			Name:            nilIfStringEmpty(&rp.Name),
			PublicSubdomain: nilIfStringEmpty(&rp.PublicSubdomain),
			FQDN:            nilIfStringEmpty(&rp.FQDN),
		}
	}

	oc.Properties.AgentPoolProfiles = make([]v20180930preview.AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles))
	for i := range cs.Properties.AgentPoolProfiles {
		app := cs.Properties.AgentPoolProfiles[i]
		vmSize := v20180930preview.VMSize(app.VMSize)
		if app.Role == AgentPoolProfileRoleMaster {
			oc.Properties.MasterPoolProfile = &v20180930preview.MasterPoolProfile{
				Count:      &app.Count,
				VMSize:     nilIfv20180930VMSizeEmpty(&vmSize),
				SubnetCIDR: nilIfStringEmpty(&app.SubnetCIDR),
			}

		} else {
			osType := v20180930preview.OSType(app.OSType)
			role := v20180930preview.AgentPoolProfileRole(app.Role)

			oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, v20180930preview.AgentPoolProfile{
				Name:       nilIfStringEmpty(&app.Name),
				Count:      &app.Count,
				VMSize:     nilIfv20180930VMSizeEmpty(&vmSize),
				SubnetCIDR: nilIfStringEmpty(&app.SubnetCIDR),
				OSType:     nilIfv20180930OSTypeEmpty(&osType),
				Role:       nilIfv20180930AgentPoolProfileRoleEmpty(&role),
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
				Kind:     nilIfStringEmpty(&provider.Kind),
				ClientID: nilIfStringEmpty(&provider.ClientID),
				Secret:   nilIfStringEmpty(&provider.Secret),
				TenantID: nilIfStringEmpty(&provider.TenantID),
			}

		default:
			panic("authProfile.identityProviders conversion failed")
		}
	}

	return oc
}
