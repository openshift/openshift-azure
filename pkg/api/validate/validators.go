package validate

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/openshift/openshift-azure/pkg/api"
)

func init() {
	var err error

	// TODO: we probably need to bite the bullet and make these configurable.
	_, clusterNetworkCIDR, err = net.ParseCIDR("10.128.0.0/14")
	if err != nil {
		panic(err)
	}

	_, serviceNetworkCIDR, err = net.ParseCIDR("172.30.0.0/16")
	if err != nil {
		panic(err)
	}
}

var validAgentPoolProfileRoles = map[api.AgentPoolProfileRole]struct{}{
	api.AgentPoolProfileRoleCompute: {},
	api.AgentPoolProfileRoleInfra:   {},
	api.AgentPoolProfileRoleMaster:  {},
}

var validRouterProfileNames = map[string]struct{}{
	"default": {},
}

func isValidHostname(h string) bool {
	return len(h) <= 255 && rxRfc1123.MatchString(h)
}

func isValidCloudAppHostname(h, location string) bool {
	if !rxCloudDomainLabel.MatchString(h) {
		return false
	}
	return strings.HasSuffix(h, "."+location+".cloudapp.azure.com") && strings.Count(h, ".") == 4
}

func isValidIPV4CIDR(cidr string) bool {
	ip, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	if ip.To4() == nil {
		return false
	}
	if net == nil || !ip.Equal(net.IP) {
		return false
	}
	return true
}

func IsValidBlobContainerName(c string) bool {
	// https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-containers--blobs--and-metadata
	if !rxBlobContainerName.MatchString(c) {
		return false
	}

	if strings.Trim(c, "-") != c || strings.Contains(c, "--") {
		return false
	}

	return true
}

func vnetContainsSubnet(vnet, subnet *net.IPNet) bool {
	vnetbits, _ := vnet.Mask.Size()
	subnetbits, _ := subnet.Mask.Size()
	if vnetbits > subnetbits {
		// e.g., vnet is a /24, subnet is a /16: vnet cannot contain subnet.
		return false
	}

	return vnet.IP.Equal(subnet.IP.Mask(vnet.Mask))
}

func validateContainerService(c *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	if c == nil {
		errs = append(errs, fmt.Errorf("openShiftManagedCluster cannot be nil"))
		return
	}

	if c.Location == "" {
		errs = append(errs, fmt.Errorf("invalid location %q", c.Location))
	}

	if c.Name == "" {
		errs = append(errs, fmt.Errorf("invalid name %q", c.Name))
	}

	errs = append(errs, validateProperties(&c.Properties, c.Location, externalOnly)...)
	return
}

func validateProperties(p *api.Properties, location string, externalOnly bool) (errs []error) {
	if p == nil {
		errs = append(errs, fmt.Errorf("properties cannot be nil"))
		return
	}

	errs = append(errs, validateProvisioningState(p.ProvisioningState)...)
	switch p.OpenShiftVersion {
	case "v3.11":
	default:
		errs = append(errs, fmt.Errorf("invalid properties.openShiftVersion %q", p.OpenShiftVersion))
	}

	if p.PublicHostname != "" { // TODO: relax after private preview (&& !isValidHostname(p.PublicHostname))
		errs = append(errs, fmt.Errorf("invalid properties.publicHostname %q", p.PublicHostname))
	}
	errs = append(errs, validateNetworkProfile(&p.NetworkProfile)...)

	errs = append(errs, validateRouterProfiles(p.RouterProfiles, location, externalOnly)...)

	errs = append(errs, validateFQDN(p, location)...)
	// we can disregard any error below because we are already going to fail
	// validation if VnetCIDR does not parse correctly.
	_, vnet, _ := net.ParseCIDR(p.NetworkProfile.VnetCIDR)
	errs = append(errs, validateAgentPoolProfiles(p.AgentPoolProfiles, vnet)...)
	errs = append(errs, validateAuthProfile(&p.AuthProfile)...)
	return
}

func validateAuthProfile(ap *api.AuthProfile) (errs []error) {
	if ap == nil {
		errs = append(errs, fmt.Errorf("properties.authProfile cannot be nil"))
		return
	}

	if len(ap.IdentityProviders) != 1 {
		errs = append(errs, fmt.Errorf("invalid properties.authProfile.identityProviders length"))
	}
	//check supported identity providers
	for _, ip := range ap.IdentityProviders {
		switch provider := ip.Provider.(type) {
		case (*api.AADIdentityProvider):
			if ip.Name != "Azure AD" {
				errs = append(errs, fmt.Errorf("invalid properties.authProfile.identityProviders name"))
			}
			if provider.Secret == "" {
				errs = append(errs, fmt.Errorf("invalid properties.authProfile.AADIdentityProvider secret %q", provider.Secret))
			}
			if provider.ClientID == "" {
				errs = append(errs, fmt.Errorf("invalid properties.authProfile.AADIdentityProvider clientId %q", provider.ClientID))
			}
			if provider.TenantID == "" {
				errs = append(errs, fmt.Errorf("invalid properties.authProfile.AADIdentityProvider tenantId %q", provider.TenantID))
			}
			if provider.CustomerAdminGroupID != nil && !isValidUUID(*provider.CustomerAdminGroupID) {
				errs = append(errs, fmt.Errorf("invalid properties.authProfile.AADIdentityProvider customerAdminGroupId %q", *provider.CustomerAdminGroupID))
			}
		}
	}
	return
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
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

		errs = append(errs, validateAgentPoolProfile(app, vnet)...)
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

func validateAgentPoolProfile(app api.AgentPoolProfile, vnet *net.IPNet) (errs []error) {
	switch app.Role {
	case api.AgentPoolProfileRoleCompute:
		switch app.Name {
		case string(api.AgentPoolProfileRoleMaster), string(api.AgentPoolProfileRoleInfra):
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
		}
		if !rxAgentPoolProfileName.MatchString(app.Name) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
		}
		if app.Count < 1 || app.Count > 20 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %d", app.Name, app.Count))
		}

	case api.AgentPoolProfileRoleInfra:
		if app.Name != string(app.Role) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
		}
		if app.Count != 2 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %d", app.Name, app.Count))
		}

	case api.AgentPoolProfileRoleMaster:
		if app.Name != string(app.Role) {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
		}
		if app.Count != 3 {
			errs = append(errs, fmt.Errorf("invalid properties.masterPoolProfile.count %d", app.Count))
		}
	}

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

func IsValidAgentPoolHostname(hostname string) bool {
	parts := strings.Split(hostname, "-")
	switch len(parts) {
	case 2: // master-XXXXXX
		if parts[0] != "master" ||
			len(parts[1]) != 6 {
			return false
		}
		_, err := strconv.ParseUint(parts[1], 36, 64)
		if err != nil {
			return false
		}

	case 3: // something-XXXXXXXXXX-XXXXXX
		if !rxAgentPoolProfileName.MatchString(parts[0]) ||
			parts[0] == "master" ||
			len(parts[1]) != 10 ||
			len(parts[2]) != 6 {
			return false
		}
		_, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return false
		}
		_, err = strconv.ParseUint(parts[2], 36, 64)
		if err != nil {
			return false
		}

	default:
		return false
	}

	return true
}

func validateFQDN(p *api.Properties, location string) (errs []error) {
	if p == nil {
		errs = append(errs, fmt.Errorf("masterProfile cannot be nil"))
		return
	}
	if p.FQDN == "" || !isValidCloudAppHostname(p.FQDN, location) {
		errs = append(errs, fmt.Errorf("invalid properties.fqdn %q", p.FQDN))
	}
	return
}

func validateNetworkProfile(np *api.NetworkProfile) (errs []error) {
	if np == nil {
		errs = append(errs, fmt.Errorf("networkProfile cannot be nil"))
		return
	}
	if np.VnetID != "" && !rxVNetID.MatchString(np.VnetID) {
		errs = append(errs, fmt.Errorf("invalid properties.networkProfile.vnetId %q", np.VnetID))
	}
	if !isValidIPV4CIDR(np.VnetCIDR) {
		errs = append(errs, fmt.Errorf("invalid properties.networkProfile.vnetCidr %q", np.VnetCIDR))
	}
	if np.PeerVnetID != nil && !rxVNetID.MatchString(*np.PeerVnetID) {
		errs = append(errs, fmt.Errorf("invalid properties.networkProfile.peerVnetId %q", *np.PeerVnetID))
	}
	return
}

func validateRouterProfiles(rps []api.RouterProfile, location string, externalOnly bool) (errs []error) {
	rpmap := map[string]api.RouterProfile{}

	for _, rp := range rps {
		if _, found := validRouterProfileNames[rp.Name]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q]", rp.Name))
		}

		if _, found := rpmap[rp.Name]; found {
			errs = append(errs, fmt.Errorf("duplicate properties.routerProfiles %q", rp.Name))
		}
		rpmap[rp.Name] = rp

		errs = append(errs, validateRouterProfile(rp, location, externalOnly)...)
	}
	if !externalOnly {
		for name := range validRouterProfileNames {
			if _, found := rpmap[name]; !found {
				errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q]", name))
			}
		}
	}
	return
}

func validateRouterProfile(rp api.RouterProfile, location string, externalOnly bool) (errs []error) {
	if rp.Name == "" {
		errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q].name %q", rp.Name, rp.Name))
	}

	// TODO: consider ensuring that PublicSubdomain is of the form
	// <string>.<location>.azmosa.io
	if rp.PublicSubdomain != "" && !isValidHostname(rp.PublicSubdomain) {
		errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q].publicSubdomain %q", rp.Name, rp.PublicSubdomain))
	}

	if !externalOnly && rp.FQDN != "" && !isValidCloudAppHostname(rp.FQDN, location) {
		errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q].fqdn %q", rp.Name, rp.FQDN))
	}

	return
}

func validateProvisioningState(ps api.ProvisioningState) (errs []error) {
	switch ps {
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
		errs = append(errs, fmt.Errorf("invalid properties.provisioningState %q", ps))
	}
	return
}

func validateVMSize(app api.AgentPoolProfile, runningUnderTest bool) (errs []error) {
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
	return errs
}

func isValidMasterAndInfraVMSize(size api.VMSize, runningUnderTest bool) bool {
	if runningUnderTest && size == api.StandardD2sV3 {
		return true
	}

	_, found := validMasterAndInfraVMSizes[size]
	return found
}

func isValidComputeVMSize(size api.VMSize, runningUnderTest bool) bool {
	if runningUnderTest && size == api.StandardD2sV3 {
		return true
	}

	_, found := validComputeVMSizes[size]
	return found
}
