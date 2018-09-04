package validate

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/go-test/deep"
	"github.com/openshift/openshift-azure/pkg/api"
)

var rxRfc1123 = regexp.MustCompile(`(?i)^` +
	`([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9])` +
	`(\.([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9]))*` +
	`$`)

var rxAgentPoolProfileVNetSubnetID = regexp.MustCompile(`(?i)^` +
	`/subscriptions/[^/]+` +
	`/resourceGroups/[^/]+` +
	`/providers/Microsoft\.Network` +
	`/virtualNetworks/[^/]+` +
	`/subnets/[^/]+` +
	`$`)

var validAgentPoolProfileNames = map[api.AgentPoolProfileRole]struct{}{
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

// ContainerService validates a ContainerService struct
func Validate(new, old *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	// TODO update validation
	// TODO are these error messages confusing since they may not correspond with the external model?
	if errs := validateContainerService(new, externalOnly); len(errs) > 0 {
		return errs
	}
	if old != nil {
		return validateUpdateContainerService(new, old, externalOnly)
	}
	return nil
}

func validateContainerService(c *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	if c.Location == "" {
		errs = append(errs, fmt.Errorf("invalid location %q", c.Location))
	}

	if c.Name == "" {
		errs = append(errs, fmt.Errorf("invalid name %q", c.Name))
	}

	if c.Properties == nil {
		errs = append(errs, fmt.Errorf("properties cannot be nil"))
		return
	}

	errs = append(errs, validateProperties(c.Properties, externalOnly)...)
	return
}

func validateUpdateContainerService(cs, oldCs *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	if !reflect.DeepEqual(cs, oldCs) {
		errs = append(errs, fmt.Errorf("invalid change %s", deep.Equal(cs, oldCs)))
	}
	return
}

func validateProperties(p *api.Properties, externalOnly bool) (errs []error) {
	errs = append(errs, validateProvisioningState(p.ProvisioningState)...)
	switch p.OpenShiftVersion {
	case "v3.10":
	default:
		errs = append(errs, fmt.Errorf("invalid properties.openShiftVersion %q", p.OpenShiftVersion))
	}

	if p.PublicHostname != "" && !isValidHostname(p.PublicHostname) {
		errs = append(errs, fmt.Errorf("invalid properties.publicHostname %q", p.PublicHostname))
	}
	if !externalOnly {
		errs = append(errs, validateRouterProfiles(p.RouterProfiles)...)
	}
	errs = append(errs, validateFQDN(p)...)
	errs = append(errs, validateAgentPoolProfiles(p.AgentPoolProfiles)...)
	errs = append(errs, validateServicePrincipalProfile(p.ServicePrincipalProfile)...)
	errs = append(errs, validateAuthProfile(p.AuthProfile)...)
	return
}

func validateAuthProfile(ap *api.AuthProfile) (errs []error) {
	if len(ap.IdentityProviders) != 1 {
		errs = append(errs, fmt.Errorf("invalid properties.authProfile.identityProviders length"))
	}
	//check supported identity providers
	for _, ip := range ap.IdentityProviders {
		switch provider := ip.Provider.(type) {
		case (*api.AADIdentityProvider):
			if provider.Secret == "" {
				errs = append(errs, fmt.Errorf("invalid properties.authProfile.AADIdentityProvider clientId %q", provider.Secret))
			}
			if provider.ClientID == "" {
				errs = append(errs, fmt.Errorf("invalid properties.authProfile.AADIdentityProvider clientId %q", provider.ClientID))
			}
		}
	}
	return
}

func validateServicePrincipalProfile(spp *api.ServicePrincipalProfile) (errs []error) {
	if spp == nil {
		errs = append(errs, fmt.Errorf("servicePrincipalProfile cannot be nil"))
		return
	}
	if spp.ClientID == "" {
		errs = append(errs, fmt.Errorf("invalid properties.servicePrincipalProfile.clientId %q", spp.ClientID))
	}

	if spp.Secret == "" {
		errs = append(errs, fmt.Errorf("invalid properties.servicePrincipalProfile.secret %q", spp.Secret))
	}

	return
}

func validateAgentPoolProfiles(apps map[api.AgentPoolProfileRole]api.AgentPoolProfile) (errs []error) {
	var vnetSubnetID *string
	for role, app := range apps {
		if _, found := validAgentPoolProfileNames[role]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", role))
		}

		if vnetSubnetID == nil {
			// careful: not the same as &app.VnetSubnetID
			vnetSubnetID = to.StringPtr(app.VnetSubnetID)
		} else if app.VnetSubnetID != *vnetSubnetID {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles.vnetSubnetID %q: all subnets must match when using vnetSubnetID", app.VnetSubnetID))
		}

		errs = append(errs, validateAgentPoolProfile(role, &app)...)
	}

	for role := range validAgentPoolProfileNames {
		if _, found := apps[role]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", role))
		}
	}

	return
}

func validateAgentPoolProfile(role api.AgentPoolProfileRole, app *api.AgentPoolProfile) (errs []error) {
	if app.Name != string(role) {
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
	}

	switch role {
	case api.AgentPoolProfileRoleCompute:
		if app.Count < 1 || app.Count > 5 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %q", app.Name, app.Count))
		}

	case api.AgentPoolProfileRoleInfra:
		if app.Count != 2 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %q", app.Name, app.Count))
		}

	case api.AgentPoolProfileRoleMaster:
		if app.Count != 3 {
			errs = append(errs, fmt.Errorf("invalid masterPoolProfile.count %d", app.Count))
		}

	default:
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].role %q", app.Name, role))
	}

	switch app.VMSize {
	case "Standard_D2s_v3",
		"Standard_D4s_v3":
	default:
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].vmSize %q", app.Name, app.VMSize))
	}

	if app.VnetSubnetID != "" && !rxAgentPoolProfileVNetSubnetID.MatchString(app.VnetSubnetID) {
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].vnetSubnetID %q", app.Name, app.VnetSubnetID))
	}

	switch app.OSType {
	case api.OSTypeLinux:
	default:
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].osType %q", app.Name, app.OSType))
	}

	return
}

func validateFQDN(p *api.Properties) (errs []error) {
	if p == nil {
		errs = append(errs, fmt.Errorf("masterProfile cannot be nil"))
	}
	if p.FQDN == "" || !isValidHostname(p.FQDN) {
		errs = append(errs, fmt.Errorf("invalid properties.fqdn %q", p.FQDN))
	}
	return
}

func validateRouterProfiles(rps []api.RouterProfile) (errs []error) {
	rpmap := map[string]api.RouterProfile{}

	for _, rp := range rps {
		if _, found := validRouterProfileNames[rp.Name]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q]", rp.Name))
		}

		if _, found := rpmap[rp.Name]; found {
			errs = append(errs, fmt.Errorf("duplicate properties.routerProfiles %q", rp.Name))
		}
		rpmap[rp.Name] = rp

		errs = append(errs, validateRouterProfile(rp)...)
	}

	for name := range validRouterProfileNames {
		if _, found := rpmap[name]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q]", name))
		}
	}

	return
}

func validateRouterProfile(rp api.RouterProfile) (errs []error) {
	if rp.Name == "" {
		errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q].name %q", rp.Name, rp.Name))
	}

	if rp.PublicSubdomain != "" && !isValidHostname(rp.PublicSubdomain) {
		errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q].publicSubdomain %q", rp.Name, rp.PublicSubdomain))
	}

	if rp.FQDN != "" && !isValidHostname(rp.FQDN) {
		errs = append(errs, fmt.Errorf("invalid properties.routerProfiles[%q].fqdn %q", rp.Name, rp.FQDN))
	}

	return
}

func validateProvisioningState(ps api.ProvisioningState) (errs []error) {
	switch ps {
	case "",
		api.Creating,
		api.Updating,
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
