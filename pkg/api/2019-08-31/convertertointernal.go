package v20190831

import (
	"errors"

	"github.com/openshift/openshift-azure/pkg/api"
)

// ToInternal converts from a
// v20190831.OpenShiftManagedCluster to an internal.OpenShiftManagedCluster.
// If old is non-nil, it is going to be used as the base for the internal
// output where the external request is merged on top of.
func ToInternal(oc *OpenShiftManagedCluster, old *api.OpenShiftManagedCluster) (*api.OpenShiftManagedCluster, error) {
	cs := &api.OpenShiftManagedCluster{}

	if old == nil {
		setDefaults(oc)
	} else {
		cs = old.DeepCopy()
	}

	if oc.ID != nil {
		cs.ID = *oc.ID
	}
	if oc.Name != nil {
		cs.Name = *oc.Name
	}
	if oc.Type != nil {
		cs.Type = *oc.Type
	}
	if oc.Location != nil {
		cs.Location = *oc.Location
	}
	if cs.Tags == nil && len(oc.Tags) > 0 {
		cs.Tags = make(map[string]string, len(oc.Tags))
	}
	for k, v := range oc.Tags {
		if v != nil {
			cs.Tags[k] = *v
		}
	}

	if oc.Plan != nil {
		mergeFromResourcePurchasePlan(oc, cs)
	}

	if err := mergeProperties(oc, cs); err != nil {
		return nil, err
	}

	return cs, nil
}

// mergeFromResourcePurchasePlan merges filled out fields from the external API to the internal representation, doesn't change fields which are nil in the input.
// The user is allowed to provide partial PurchasePlan via the API.
func mergeFromResourcePurchasePlan(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) {
	if cs.Plan == nil {
		cs.Plan = &api.ResourcePurchasePlan{}
	}
	if oc.Plan.Name != nil {
		cs.Plan.Name = oc.Plan.Name
	}
	if oc.Plan.Product != nil {
		cs.Plan.Product = oc.Plan.Product
	}
	if oc.Plan.PromotionCode != nil {
		cs.Plan.PromotionCode = oc.Plan.PromotionCode
	}
	if oc.Plan.Publisher != nil {
		cs.Plan.Publisher = oc.Plan.Publisher
	}
}

func mergeProperties(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if oc.Properties == nil {
		return nil
	}
	if oc.Properties.ProvisioningState != nil {
		cs.Properties.ProvisioningState = api.ProvisioningState(*oc.Properties.ProvisioningState)
	}
	if oc.Properties.OpenShiftVersion != nil {
		cs.Properties.OpenShiftVersion = *oc.Properties.OpenShiftVersion
	}
	if oc.Properties.ClusterVersion != nil {
		cs.Properties.ClusterVersion = *oc.Properties.ClusterVersion
	}
	if oc.Properties.PublicHostname != nil {
		cs.Properties.PublicHostname = *oc.Properties.PublicHostname
	}
	if oc.Properties.FQDN != nil {
		cs.Properties.FQDN = *oc.Properties.FQDN
	}

	if oc.Properties.NetworkProfile != nil {
		if oc.Properties.NetworkProfile.VnetID != nil {
			cs.Properties.NetworkProfile.VnetID = *oc.Properties.NetworkProfile.VnetID
		}
		if oc.Properties.NetworkProfile.VnetCIDR != nil {
			cs.Properties.NetworkProfile.VnetCIDR = *oc.Properties.NetworkProfile.VnetCIDR
		}
		cs.Properties.NetworkProfile.PeerVnetID = oc.Properties.NetworkProfile.PeerVnetID
	}

	if oc.Properties.MonitorProfile != nil {
		if oc.Properties.MonitorProfile.WorkspaceResourceID != nil {
			cs.Properties.MonitorProfile.WorkspaceResourceID = *oc.Properties.MonitorProfile.WorkspaceResourceID
		}
	}

	if err := mergeRouterProfiles(oc, cs); err != nil {
		return err
	}

	if err := mergeAgentPoolProfiles(oc, cs); err != nil {
		return err
	}

	return mergeAuthProfile(oc, cs)
}

func mergeRouterProfiles(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if cs.Properties.RouterProfiles == nil && len(oc.Properties.RouterProfiles) > 0 {
		cs.Properties.RouterProfiles = make([]api.RouterProfile, 0, len(oc.Properties.RouterProfiles))
	}
	for _, rp := range oc.Properties.RouterProfiles {
		if rp.Name == nil || *rp.Name == "" {
			return errors.New("invalid router profile - name is missing")
		}

		index := routerProfileIndex(*rp.Name, cs.Properties.RouterProfiles)
		// If the requested profile does not exist, add it
		// in cs as is, otherwise merge it in the existing
		// profile.
		if index == -1 {
			cs.Properties.RouterProfiles = append(cs.Properties.RouterProfiles, convertRouterProfile(rp, nil))
		} else {
			head := append(cs.Properties.RouterProfiles[:index], convertRouterProfile(rp, &cs.Properties.RouterProfiles[index]))
			cs.Properties.RouterProfiles = append(head, cs.Properties.RouterProfiles[index+1:]...)
		}
	}
	return nil
}

func routerProfileIndex(name string, profiles []api.RouterProfile) int {
	for i, profile := range profiles {
		if profile.Name == name {
			return i
		}
	}
	return -1
}

func convertRouterProfile(in RouterProfile, old *api.RouterProfile) (out api.RouterProfile) {
	if old != nil {
		out = *old
	}
	if in.Name != nil {
		out.Name = *in.Name
	}
	if in.PublicSubdomain != nil {
		out.PublicSubdomain = *in.PublicSubdomain
	}
	if in.FQDN != nil {
		out.FQDN = *in.FQDN
	}
	return
}

func mergeAgentPoolProfiles(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if cs.Properties.AgentPoolProfiles == nil && len(oc.Properties.AgentPoolProfiles) > 0 {
		cs.Properties.AgentPoolProfiles = make([]api.AgentPoolProfile, 0, len(oc.Properties.AgentPoolProfiles)+1)
	}

	if p := oc.Properties.MasterPoolProfile; p != nil {
		index := agentPoolProfileIndex(string(api.AgentPoolProfileRoleMaster), cs.Properties.AgentPoolProfiles)
		// the master profile does not exist, add it as is
		if index == -1 {
			cs.Properties.AgentPoolProfiles = append(cs.Properties.AgentPoolProfiles, convertMasterToAgentPoolProfile(*p, nil))
		} else {
			head := append(cs.Properties.AgentPoolProfiles[:index], convertMasterToAgentPoolProfile(*p, &cs.Properties.AgentPoolProfiles[index]))
			cs.Properties.AgentPoolProfiles = append(head, cs.Properties.AgentPoolProfiles[index+1:]...)
		}
	}

	for _, in := range oc.Properties.AgentPoolProfiles {
		if in.Name == nil || *in.Name == "" {
			return errors.New("invalid agent pool profile - name is missing")
		}
		index := agentPoolProfileIndex(*in.Name, cs.Properties.AgentPoolProfiles)
		// If the requested profile does not exist, add it
		// in cs as is, otherwise merge it in the existing
		// profile.
		if index == -1 {
			cs.Properties.AgentPoolProfiles = append(cs.Properties.AgentPoolProfiles, convertAgentPoolProfile(in, nil))
		} else {
			head := append(cs.Properties.AgentPoolProfiles[:index], convertAgentPoolProfile(in, &cs.Properties.AgentPoolProfiles[index]))
			cs.Properties.AgentPoolProfiles = append(head, cs.Properties.AgentPoolProfiles[index+1:]...)
		}
	}
	return nil
}

func agentPoolProfileIndex(name string, profiles []api.AgentPoolProfile) int {
	for i, profile := range profiles {
		if profile.Name == name {
			return i
		}
	}
	return -1
}

func convertMasterToAgentPoolProfile(in MasterPoolProfile, old *api.AgentPoolProfile) (out api.AgentPoolProfile) {
	if old != nil {
		out = *old
	}
	out.Name = string(api.AgentPoolProfileRoleMaster)
	out.Role = api.AgentPoolProfileRoleMaster
	out.OSType = api.OSTypeLinux
	if in.Count != nil {
		out.Count = *in.Count
	}
	if in.VMSize != nil {
		out.VMSize = api.VMSize(*in.VMSize)
	}
	if in.SubnetCIDR != nil {
		out.SubnetCIDR = *in.SubnetCIDR
	}
	return
}

func convertAgentPoolProfile(in AgentPoolProfile, old *api.AgentPoolProfile) (out api.AgentPoolProfile) {
	if old != nil {
		out = *old
	}
	if in.Name != nil {
		out.Name = *in.Name
	}
	if in.Count != nil {
		out.Count = *in.Count
	}
	if in.VMSize != nil {
		out.VMSize = api.VMSize(*in.VMSize)
	}
	if in.SubnetCIDR != nil {
		out.SubnetCIDR = *in.SubnetCIDR
	}
	if in.OSType != nil {
		out.OSType = api.OSType(*in.OSType)
	}
	if in.Role != nil {
		out.Role = api.AgentPoolProfileRole(*in.Role)
	}
	return
}

func mergeAuthProfile(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if oc.Properties.AuthProfile == nil {
		return nil
	}

	if cs.Properties.AuthProfile.IdentityProviders == nil && len(oc.Properties.AuthProfile.IdentityProviders) > 0 {
		cs.Properties.AuthProfile.IdentityProviders = make([]api.IdentityProvider, 0, len(oc.Properties.AuthProfile.IdentityProviders))
	}

	for _, ip := range oc.Properties.AuthProfile.IdentityProviders {
		if ip.Name == nil || *ip.Name == "" {
			return errors.New("invalid identity provider - name is missing")
		}
		index := identityProviderIndex(*ip.Name, cs.Properties.AuthProfile.IdentityProviders)
		// If the requested provider does not exist, add it
		// in cs as is, otherwise merge it in the existing
		// provider.
		if index == -1 {
			cs.Properties.AuthProfile.IdentityProviders = append(cs.Properties.AuthProfile.IdentityProviders, convertIdentityProvider(ip, nil))
		} else {
			provider := cs.Properties.AuthProfile.IdentityProviders[index].Provider
			switch out := provider.(type) {
			case (*api.AADIdentityProvider):
				in := ip.Provider.(*AADIdentityProvider)
				if in.Kind != nil {
					if out.Kind != "" && out.Kind != *in.Kind {
						return errors.New("cannot update the kind of the identity provider")
					}
				}
			default:
				return errors.New("authProfile.identityProviders conversion failed")
			}
			head := append(cs.Properties.AuthProfile.IdentityProviders[:index], convertIdentityProvider(ip, &cs.Properties.AuthProfile.IdentityProviders[index]))
			cs.Properties.AuthProfile.IdentityProviders = append(head, cs.Properties.AuthProfile.IdentityProviders[index+1:]...)
		}
	}
	return nil
}

func identityProviderIndex(name string, providers []api.IdentityProvider) int {
	for i, provider := range providers {
		if provider.Name == name {
			return i
		}
	}
	return -1
}

func convertIdentityProvider(in IdentityProvider, old *api.IdentityProvider) (out api.IdentityProvider) {
	if old != nil {
		out = *old
	}
	if in.Name != nil {
		out.Name = *in.Name
	}
	if in.Provider != nil {
		switch provider := in.Provider.(type) {
		case (*AADIdentityProvider):
			p := &api.AADIdentityProvider{}
			if out.Provider != nil {
				p = out.Provider.(*api.AADIdentityProvider)
			}
			if provider.Kind != nil {
				p.Kind = *provider.Kind
			}
			if provider.ClientID != nil {
				p.ClientID = *provider.ClientID
			}
			if provider.Secret != nil {
				p.Secret = *provider.Secret
			}
			if provider.TenantID != nil {
				p.TenantID = *provider.TenantID
			}
			p.CustomerAdminGroupID = provider.CustomerAdminGroupID
			out.Provider = p

		default:
			panic("authProfile.identityProviders conversion failed")
		}
	}
	return
}
