package converter

import (
	"github.com/openshift/openshift-azure/pkg/api"
	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30/api"
)

// ConvertTov20190430 converts from an OpenShiftManagedCluster to a
// v20190430.OpenShiftManagedCluster.
func ConvertTov20190430(cs *api.OpenShiftManagedCluster) *v20190430.OpenShiftManagedCluster {
	oc := &v20190430.OpenShiftManagedCluster{
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
		oc.Plan = &v20190430.ResourcePurchasePlan{
			Name:          cs.Plan.Name,
			Product:       cs.Plan.Product,
			PromotionCode: cs.Plan.PromotionCode,
			Publisher:     cs.Plan.Publisher,
		}
	}

	provisioningState := v20190430.ProvisioningState(cs.Properties.ProvisioningState)
	oc.Properties = &v20190430.Properties{
		ProvisioningState: &provisioningState,
		OpenShiftVersion:  &cs.Properties.OpenShiftVersion,
		ClusterVersion:    &cs.Properties.ClusterVersion,
		PublicHostname:    &cs.Properties.PublicHostname,
		FQDN:              &cs.Properties.FQDN,
	}

	oc.Properties.NetworkProfile = &v20190430.NetworkProfile{
		VnetID:     &cs.Properties.NetworkProfile.VnetID,
		VnetCIDR:   &cs.Properties.NetworkProfile.VnetCIDR,
		PeerVnetID: cs.Properties.NetworkProfile.PeerVnetID,
	}

	oc.Properties.RouterProfiles = make([]v20190430.RouterProfile, len(cs.Properties.RouterProfiles))
	for i := range cs.Properties.RouterProfiles {
		rp := cs.Properties.RouterProfiles[i]
		oc.Properties.RouterProfiles[i] = v20190430.RouterProfile{
			Name:            &rp.Name,
			PublicSubdomain: &rp.PublicSubdomain,
			FQDN:            &rp.FQDN,
		}
	}

	oc.Properties.AgentPoolProfiles = make([]v20190430.AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles))
	for i := range cs.Properties.AgentPoolProfiles {
		app := cs.Properties.AgentPoolProfiles[i]
		vmSize := v20190430.VMSize(app.VMSize)

		if app.Role == api.AgentPoolProfileRoleMaster {
			oc.Properties.MasterPoolProfile = &v20190430.MasterPoolProfile{
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
			}
		} else {
			osType := v20190430.OSType(app.OSType)
			role := v20190430.AgentPoolProfileRole(app.Role)

			oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, v20190430.AgentPoolProfile{
				Name:       &app.Name,
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
				OSType:     &osType,
				Role:       &role,
			})
		}
	}

	oc.Properties.AuthProfile = &v20190430.AuthProfile{}
	oc.Properties.AuthProfile.IdentityProviders = make([]v20190430.IdentityProvider, len(cs.Properties.AuthProfile.IdentityProviders))
	for i := range cs.Properties.AuthProfile.IdentityProviders {
		ip := cs.Properties.AuthProfile.IdentityProviders[i]
		oc.Properties.AuthProfile.IdentityProviders[i].Name = &ip.Name
		switch provider := ip.Provider.(type) {
		case (*api.AADIdentityProvider):
			oc.Properties.AuthProfile.IdentityProviders[i].Provider = &v20190430.AADIdentityProvider{
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
