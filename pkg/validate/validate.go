package validate

import (
	"fmt"
	"regexp"

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

var validAgentPoolProfileNames = map[string]struct{}{
	string(api.AgentPoolProfileRoleCompute): struct{}{},
	string(api.AgentPoolProfileRoleInfra):   struct{}{},
	string(api.AgentPoolProfileRoleMaster):  struct{}{},
}

var validRouterProfileNames = map[string]struct{}{
	"default": struct{}{},
}

func isValidHostname(h string) bool {
	return len(h) <= 255 && rxRfc1123.MatchString(h)
}

// ContainerService validates a ContainerService struct
func ContainerService(new, old *api.OpenShiftManagedCluster) (errs []error) {
	// TODO update validation
	// TODO are these error messages confusing since they may not correspond with the external model?
	return validateContainerService(new)
}

func validateContainerService(c *api.OpenShiftManagedCluster) (errs []error) {
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

	errs = append(errs, validateProperties(c.Properties)...)
	return
}

func validateProperties(p *api.Properties) (errs []error) {
	errs = append(errs, validateProvisioningState(p.ProvisioningState)...)
	errs = append(errs, validateOrchestratorProfile(p.OrchestratorProfile)...)
	errs = append(errs, validateFQDN(p)...)
	errs = append(errs, validateAgentPoolProfiles(p.AgentPoolProfiles)...)
	errs = append(errs, validateServicePrincipalProfile(p.ServicePrincipalProfile)...)
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

func validateAgentPoolProfiles(apps []*api.AgentPoolProfile) (errs []error) {
	appmap := map[string]*api.AgentPoolProfile{}

	for i, app := range apps {
		// TODO why is this a pointer?
		if app == nil {
			continue
		}

		if _, found := validAgentPoolProfileNames[app.Name]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", app.Name))
		}

		if _, found := appmap[app.Name]; found {
			errs = append(errs, fmt.Errorf("duplicate properties.agentPoolProfiles %q", app.Name))
		}
		appmap[app.Name] = app

		if i > 0 && app.VnetSubnetID != apps[i-1].VnetSubnetID {
			errs = append(errs, fmt.Errorf("duplicate properties.agentPoolProfiles.vnetSubnetID %q", app.VnetSubnetID))
		}

		errs = append(errs, validateAgentPoolProfile(app)...)
	}

	for name := range validAgentPoolProfileNames {
		if _, found := appmap[name]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", name))
		}
	}

	return
}

func validateAgentPoolProfile(app *api.AgentPoolProfile) (errs []error) {
	if app.Name != string(app.Role) {
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
	}

	switch app.Role {
	case api.AgentPoolProfileRoleCompute,
		api.AgentPoolProfileRoleInfra:
		if app.Count < 1 || app.Count > 100 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %q", app.Name, app.Count))
		}

	case api.AgentPoolProfileRoleMaster:
		if app.Count < 3 {
			errs = append(errs, fmt.Errorf("invalid masterPoolProfile.count %d", app.Count))
		}

	default:
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].role %q", app.Name, app.Role))
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

func validateOrchestratorProfile(p *api.OrchestratorProfile) (errs []error) {
	if p == nil {
		errs = append(errs, fmt.Errorf("orchestratorProfile cannot be nil"))
		return
	}

	switch p.OrchestratorVersion {
	case "v3.10":
	default:
		errs = append(errs, fmt.Errorf("invalid properties.openShiftVersion %q", p.OrchestratorVersion))
	}

	errs = append(errs, validateOpenShiftConfig(p.OpenShiftConfig)...)

	return
}

func validateFQDN(p *api.Properties) (errs []error) {
	if p == nil {
		errs = append(errs, fmt.Errorf("masterProfile cannot be nil"))
	}
	if p.FQDN != "" && !isValidHostname(p.FQDN) {
		errs = append(errs, fmt.Errorf("invalid properties.fqdn %q", p.FQDN))
	}
	return
}

func validateOpenShiftConfig(c *api.OpenShiftConfig) (errs []error) {
	if c == nil {
		errs = append(errs, fmt.Errorf("openshiftConfig cannot be nil"))
		return
	}

	if c.PublicHostname != "" && !isValidHostname(c.PublicHostname) {
		errs = append(errs, fmt.Errorf("invalid properties.publicHostname %q", c.PublicHostname))
	}
	errs = append(errs, validateRouterProfiles(c.RouterProfiles)...)

	return
}

func validateRouterProfiles(rps []api.OpenShiftRouterProfile) (errs []error) {
	rpmap := map[string]api.OpenShiftRouterProfile{}

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

func validateRouterProfile(rp api.OpenShiftRouterProfile) (errs []error) {
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
