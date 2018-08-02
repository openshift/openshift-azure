package validate

import (
	"fmt"
	"regexp"

	"github.com/openshift/openshift-azure/pkg/api/v1"
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
	string(v1.AgentPoolProfileRoleCompute): struct{}{},
	string(v1.AgentPoolProfileRoleInfra):   struct{}{},
	string(v1.AgentPoolProfileRoleMaster):  struct{}{},
}

var validRouterProfileNames = map[string]struct{}{
	"default": struct{}{},
}

func isValidHostname(h string) bool {
	return len(h) <= 255 && rxRfc1123.MatchString(h)
}

// OpenShiftCluster validates an OpenShiftCluster struct
func OpenShiftCluster(oc *v1.OpenShiftCluster) (errs []error) {
	if oc.Location == "" {
		errs = append(errs, fmt.Errorf("invalid location %q", oc.Location))
	}

	if oc.Name == "" {
		errs = append(errs, fmt.Errorf("invalid name %q", oc.Name))
	}

	if oc.Properties != nil {
		errs = append(errs, validateProperties(oc.Properties)...)
	}

	return
}

func validateProperties(p *v1.Properties) (errs []error) {
	switch p.ProvisioningState {
	case "",
		v1.Creating,
		v1.Updating,
		v1.Failed,
		v1.Succeeded,
		v1.Deleting,
		v1.Migrating,
		v1.Upgrading:
	default:
		errs = append(errs, fmt.Errorf("invalid properties.provisioningState %q", p.ProvisioningState))
	}

	switch p.OpenShiftVersion {
	case "v3.10":
	default:
		errs = append(errs, fmt.Errorf("invalid properties.openShiftVersion %q", p.OpenShiftVersion))
	}

	if p.PublicHostname != "" && !isValidHostname(p.PublicHostname) {
		errs = append(errs, fmt.Errorf("invalid properties.publicHostname %q", p.PublicHostname))
	}

	if p.FQDN != "" && !isValidHostname(p.FQDN) {
		errs = append(errs, fmt.Errorf("invalid properties.fqdn %q", p.FQDN))
	}

	errs = append(errs, validateRouterProfiles(p.RouterProfiles)...)

	errs = append(errs, validateAgentPoolProfiles(p.AgentPoolProfiles)...)

	errs = append(errs, validateServicePrincipalProfile(&p.ServicePrincipalProfile)...)

	return
}

func validateRouterProfiles(rps []v1.RouterProfile) (errs []error) {
	rpmap := map[string]v1.RouterProfile{}

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

func validateRouterProfile(rp v1.RouterProfile) (errs []error) {
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

func validateAgentPoolProfiles(apps []v1.AgentPoolProfile) (errs []error) {
	appmap := map[string]v1.AgentPoolProfile{}

	for i, app := range apps {
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

		errs = append(errs, validateAgentPoolProfile(&app)...)
	}

	for name := range validAgentPoolProfileNames {
		if _, found := appmap[name]; !found {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q]", name))
		}
	}

	return
}

func validateAgentPoolProfile(app *v1.AgentPoolProfile) (errs []error) {
	if app.Name != string(app.Role) {
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].name %q", app.Name, app.Name))
	}

	switch app.Role {
	case v1.AgentPoolProfileRoleCompute,
		v1.AgentPoolProfileRoleInfra:
		if app.Count < 1 || app.Count > 100 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %q", app.Name, app.Count))
		}

	case v1.AgentPoolProfileRoleMaster:
		if app.Count != 3 {
			errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].count %q", app.Name, app.Count))
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
	case v1.OSTypeLinux:
	default:
		errs = append(errs, fmt.Errorf("invalid properties.agentPoolProfiles[%q].osType %q", app.Name, app.OSType))
	}

	return
}

func validateServicePrincipalProfile(spp *v1.ServicePrincipalProfile) (errs []error) {
	if spp.ClientID == "" {
		errs = append(errs, fmt.Errorf("invalid properties.servicePrincipalProfile.clientId %q", spp.ClientID))
	}

	if spp.Secret == "" {
		errs = append(errs, fmt.Errorf("invalid properties.servicePrincipalProfile.secret %q", spp.Secret))
	}

	return
}
