package validate

import (
	"fmt"
	"net"

	"github.com/openshift/openshift-azure/pkg/api"
)

func validateContainerService(c *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	if c == nil {
		errs = append(errs, fmt.Errorf("openShiftManagedCluster cannot be nil"))
		return
	}

	errs = append(errs, validateProperties(&c.Properties, c.Location, externalOnly)...)

	if !isValidClusterName(c.Name) {
		errs = append(errs, fmt.Errorf("invalid name %q", c.Name))
	}

	if !isValidLocation(c.Location) {
		errs = append(errs, fmt.Errorf("invalid location %q", c.Location))
	}

	// TODO: we don't currently validate Config

	return
}

func validateProperties(p *api.Properties, location string, externalOnly bool) (errs []error) {
	if p == nil {
		errs = append(errs, fmt.Errorf("properties cannot be nil"))
		return
	}

	switch p.ProvisioningState {
	case "",
		api.Creating,
		api.Updating,
		api.AdminUpdating,
		api.Failed,
		api.Succeeded,
		api.Deleting,
		api.Migrating,
		api.Upgrading:
	default:
		errs = append(errs, fmt.Errorf("invalid properties.provisioningState %q", p.ProvisioningState))
	}

	switch p.OpenShiftVersion {
	case "v3.11":
	default:
		errs = append(errs, fmt.Errorf("invalid properties.openShiftVersion %q", p.OpenShiftVersion))
	}

	if !externalOnly {
		// TODO: consider ensuring that PublicSubdomain is of the form
		// openshift.<random>.<location>.azmosa.io
		if !isValidHostname(p.PublicHostname) {
			errs = append(errs, fmt.Errorf("invalid properties.publicHostname %q", p.PublicHostname))
		}

		if !isValidCloudAppHostname(p.FQDN, location) {
			errs = append(errs, fmt.Errorf("invalid properties.fqdn %q", p.FQDN))
		}

		if p.FQDN == p.PublicHostname {
			errs = append(errs, fmt.Errorf("invalid properties.fqdn %q: must differ from properties.publicHostname", p.FQDN))
		}
	}

	errs = append(errs, validateNetworkProfile("properties.networkProfile", &p.NetworkProfile)...)

	errs = append(errs, validateRouterProfiles("properties.routerProfiles", p.RouterProfiles, location, externalOnly)...)

	// we can disregard any error below because we are already going to fail
	// validation if VnetCIDR does not parse correctly.
	_, vnet, _ := net.ParseCIDR(p.NetworkProfile.VnetCIDR)

	errs = append(errs, validateAgentPoolProfiles(p.AgentPoolProfiles, vnet)...)

	errs = append(errs, validateAuthProfile("properties.authProfile", &p.AuthProfile)...)

	if !externalOnly {
		errs = append(errs, validateServicePrincipalProfile("properties.masterServicePrincipalProfile", &p.MasterServicePrincipalProfile)...)

		errs = append(errs, validateServicePrincipalProfile("properties.workerServicePrincipalProfile", &p.WorkerServicePrincipalProfile)...)

		errs = append(errs, validateAzProfile("properties.azProfile", &p.AzProfile)...)

		errs = append(errs, validateCertProfile("properties.apiCertProfile", &p.APICertProfile)...)
	}

	return
}

func validateNetworkProfile(path string, np *api.NetworkProfile) (errs []error) {
	if np == nil {
		errs = append(errs, fmt.Errorf("%s cannot be nil", path))
		return
	}

	if !isValidIPV4CIDR(np.VnetCIDR) {
		errs = append(errs, fmt.Errorf("invalid %s.vnetCidr %q", path, np.VnetCIDR))
	}

	if np.VnetID != "" && !rxVNetID.MatchString(np.VnetID) {
		errs = append(errs, fmt.Errorf("invalid %s.vnetId %q", path, np.VnetID))
	}

	if np.PeerVnetID != nil && !rxVNetID.MatchString(*np.PeerVnetID) {
		errs = append(errs, fmt.Errorf("invalid %s.peerVnetId %q", path, *np.PeerVnetID))
	}

	return
}

func validateRouterProfiles(path string, rps []api.RouterProfile, location string, externalOnly bool) (errs []error) {
	rpmap := map[string]api.RouterProfile{}

	for _, rp := range rps {
		if _, found := validRouterProfileNames[rp.Name]; !found {
			errs = append(errs, fmt.Errorf("invalid %s[%q]", path, rp.Name))
		}

		if _, found := rpmap[rp.Name]; found {
			errs = append(errs, fmt.Errorf("duplicate %s[%q]", path, rp.Name))
		}
		rpmap[rp.Name] = rp

		errs = append(errs, validateRouterProfile(fmt.Sprintf("%s[%q]", path, rp.Name), &rp, location, externalOnly)...)
	}

	// a bit questionable: seems that we allow a user to PUT empty
	// RouterProfiles on cluster creation.  We allow this on the first
	// validation pass (externalOnly set), then the RP apparently defaults in a
	// RouterProfile before validating a second time without externalOnly set.
	if !externalOnly {
		for name := range validRouterProfileNames {
			if _, found := rpmap[name]; !found {
				errs = append(errs, fmt.Errorf("invalid %s[%q]", path, name))
			}
		}
	}

	return
}

func validateRouterProfile(path string, rp *api.RouterProfile, location string, externalOnly bool) (errs []error) {
	if rp == nil {
		errs = append(errs, fmt.Errorf("%s cannot be nil", path))
		return
	}

	if !rxRouterProfileName.MatchString(rp.Name) {
		errs = append(errs, fmt.Errorf("invalid %s", path))
	}

	if !externalOnly {
		// TODO: consider ensuring that PublicSubdomain is of the form
		// apps.<random>.<location>.azmosa.io
		if !isValidHostname(rp.PublicSubdomain) {
			errs = append(errs, fmt.Errorf("invalid %s.publicSubdomain %q", path, rp.PublicSubdomain))
		}

		if !isValidCloudAppHostname(rp.FQDN, location) {
			errs = append(errs, fmt.Errorf("invalid %s.fqdn %q", path, rp.FQDN))
		}

		if rp.FQDN == rp.PublicSubdomain {
			errs = append(errs, fmt.Errorf("invalid %s.fqdn %q: must differ from %[1]s.publicSubdomain", path, rp.FQDN))
		}

		errs = append(errs, validateCertProfile(path+".routerCertProfile", &rp.RouterCertProfile)...)
	}

	return
}

func validateCertProfile(path string, cp *api.CertProfile) (errs []error) {
	if cp == nil {
		errs = append(errs, fmt.Errorf("%s cannot be nil", path))
		return
	}

	if !rxKeyVaultSecretURL.MatchString(cp.KeyVaultSecretURL) {
		errs = append(errs, fmt.Errorf("invalid %s.keyVaultSecretURL %q", path, cp.KeyVaultSecretURL))
	}

	return
}

func validateAgentPoolProfiles(apps []api.AgentPoolProfile, vnet *net.IPNet) (errs []error) {
	appmap := map[api.AgentPoolProfileRole]api.AgentPoolProfile{}

	for i, app := range apps {
		if _, found := validAgentPoolProfileRoles[app.Role]; !found {
			errs = append(errs, fmt.Errorf("invalid role %q in properties.agentPoolProfiles[%q]", app.Role, app.Name))
		}

		if _, found := appmap[app.Role]; found {
			errs = append(errs, fmt.Errorf("duplicate role %q in properties.agentPoolProfiles[%q]", app.Role, app.Name))
		}
		appmap[app.Role] = app

		if i > 0 && app.SubnetCIDR != apps[i-1].SubnetCIDR { // TODO: in the future, test that these are disjoint
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles.subnetCidr %q: all subnetCidrs must match", app.SubnetCIDR))
		}

		errs = append(errs, validateAgentPoolProfile(&app, vnet)...)
	}

	for role := range validAgentPoolProfileRoles {
		if _, found := appmap[role]; !found {
			errs = append(errs, fmt.Errorf("missing role %q in properties.agentPoolProfiles", role))
		}
	}

	if appmap[api.AgentPoolProfileRoleMaster].VMSize != appmap[api.AgentPoolProfileRoleInfra].VMSize {
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles.vmSize %q: master and infra vmSizes must match", appmap[api.AgentPoolProfileRoleInfra].VMSize))
	}

	return
}

func validateAgentPoolProfile(app *api.AgentPoolProfile, vnet *net.IPNet) (errs []error) {
	if app == nil {
		errs = append(errs, fmt.Errorf("agentPoolProfile cannot be nil"))
		return
	}

	switch app.Role {
	case api.AgentPoolProfileRoleCompute:
		switch app.Name {
		case string(api.AgentPoolProfileRoleMaster), string(api.AgentPoolProfileRoleInfra):
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", app.Name))
		}
		if !rxAgentPoolProfileName.MatchString(app.Name) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
		}
		if app.Count < 1 || app.Count > 30 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %d", app.Name, app.Count))
		}

	case api.AgentPoolProfileRoleInfra:
		if app.Name != string(app.Role) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", app.Name))
		}
		if app.Count != 3 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %d", app.Name, app.Count))
		}

	case api.AgentPoolProfileRoleMaster:
		if app.Name != string(app.Role) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", app.Name))
		}
		if app.Count != 3 {
			errs = append(errs, fmt.Errorf("invalid properties.masterPoolProfile.count %d", app.Count))
		}
	}

	// VMSize is checked by the caller using validateVMSize as it depends on
	// runningUnderTest.

	if !isValidIPV4CIDR(app.SubnetCIDR) {
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].subnetCidr %q", app.Name, app.SubnetCIDR))
	}

	_, subnet, _ := net.ParseCIDR(app.SubnetCIDR)
	if vnet != nil && subnet != nil {
		// we are already going to fail validation if one of these is nil.

		if !vnetContainsSubnet(vnet, subnet) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].subnetCidr %q: not contained in properties.networkProfile.vnetCidr %q", app.Name, app.SubnetCIDR, vnet.String()))
		}
		if vnetContainsSubnet(serviceNetworkCIDR, subnet) || vnetContainsSubnet(subnet, serviceNetworkCIDR) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].subnetCidr %q: overlaps with service network %q", app.Name, app.SubnetCIDR, serviceNetworkCIDR.String()))
		}
		if vnetContainsSubnet(clusterNetworkCIDR, subnet) || vnetContainsSubnet(subnet, clusterNetworkCIDR) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].subnetCidr %q: overlaps with cluster network %q", app.Name, app.SubnetCIDR, clusterNetworkCIDR.String()))
		}
	}

	switch app.OSType {
	case api.OSTypeLinux:
	default:
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].osType %q", app.Name, app.OSType))
	}

	return
}

func validateAuthProfile(path string, ap *api.AuthProfile) (errs []error) {
	if ap == nil {
		errs = append(errs, fmt.Errorf("%s cannot be nil", path))
		return
	}

	if len(ap.IdentityProviders) != 1 {
		errs = append(errs, fmt.Errorf("invalid %s.identityProviders length", path))
	}

	//check supported identity providers
	for _, ip := range ap.IdentityProviders {
		if ip.Name != "Azure AD" {
			errs = append(errs, fmt.Errorf("invalid %s.identityProviders[%q]", path, ip.Name))
		}
		switch provider := ip.Provider.(type) {
		case (*api.AADIdentityProvider):
			if provider.Kind != "AADIdentityProvider" {
				errs = append(errs, fmt.Errorf("invalid %s.identityProviders[%q].kind %q", path, ip.Name, provider.Kind))
			}
			if !isValidUUID(provider.ClientID) {
				errs = append(errs, fmt.Errorf("invalid %s.identityProviders[%q].clientId %q", path, ip.Name, provider.ClientID))
			}
			if provider.Secret == "" {
				errs = append(errs, fmt.Errorf("invalid %s.identityProviders[%q].secret %q", path, ip.Name, "<hidden>"))
			}
			if !isValidUUID(provider.TenantID) {
				errs = append(errs, fmt.Errorf("invalid %s.identityProviders[%q].tenantId %q", path, ip.Name, provider.TenantID))
			}
			if provider.CustomerAdminGroupID != nil && !isValidUUID(*provider.CustomerAdminGroupID) {
				errs = append(errs, fmt.Errorf("invalid %s.identityProviders[%q].customerAdminGroupId %q", path, ip.Name, *provider.CustomerAdminGroupID))
			}
		}
	}
	return
}

func validateServicePrincipalProfile(path string, p *api.ServicePrincipalProfile) (errs []error) {
	if p == nil {
		errs = append(errs, fmt.Errorf("%s cannot be nil", path))
		return
	}

	if !isValidUUID(p.ClientID) {
		errs = append(errs, fmt.Errorf("invalid %s.clientID %q", path, p.ClientID))
	}

	if p.Secret == "" {
		errs = append(errs, fmt.Errorf("invalid %s.secret %q", path, p.Secret))
	}

	return
}

func validateAzProfile(path string, a *api.AzProfile) (errs []error) {
	if a == nil {
		errs = append(errs, fmt.Errorf("%s cannot be nil", path))
		return
	}

	if !isValidUUID(a.TenantID) {
		errs = append(errs, fmt.Errorf("invalid %s.tenantId %q", path, a.TenantID))
	}

	if !isValidUUID(a.SubscriptionID) {
		errs = append(errs, fmt.Errorf("invalid %s.subscriptionId %q", path, a.SubscriptionID))
	}

	if !rxResourceGroupName.MatchString(a.ResourceGroup) {
		errs = append(errs, fmt.Errorf("invalid %s.resourceGroup %q", path, a.ResourceGroup))
	}

	return
}

func validateVMSize(app *api.AgentPoolProfile, runningUnderTest bool) (errs []error) {
	if app == nil {
		errs = append(errs, fmt.Errorf("agentPoolProfile cannot be nil"))
		return
	}

	switch app.Role {
	case api.AgentPoolProfileRoleMaster:
		if !isValidMasterAndInfraVMSize(app.VMSize, runningUnderTest) {
			errs = append(errs, fmt.Errorf("invalid properties.masterPoolProfile.vmSize %q", app.VMSize))
		}
	case api.AgentPoolProfileRoleInfra:
		if !isValidMasterAndInfraVMSize(app.VMSize, runningUnderTest) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].vmSize %q", app.Name, app.VMSize))
		}
	case api.AgentPoolProfileRoleCompute:
		if !isValidComputeVMSize(app.VMSize, runningUnderTest) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].vmSize %q", app.Name, app.VMSize))
		}
	}

	return
}
