package vlabs

import (
	"fmt"
	"regexp"
)

var regexRfc1123 = regexp.MustCompile(`(?i)` +
	`^([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9])` +
	`(\.([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9]))*$`)

func isValidHostname(h string) bool {
	return len(h) <= 255 && regexRfc1123.MatchString(h)
}

var regexAgentPoolName = regexp.MustCompile(`^[a-z][a-z0-9]{0,11}$`)

// Validate validates an OpenShiftCluster struct
func (oc *OpenShiftCluster) Validate() error {
	if oc.Location == "" {
		return fmt.Errorf("location must not be empty")
	}
	if oc.Name == "" {
		return fmt.Errorf("name must not be empty")
	}

	return oc.Properties.Validate()
}

// Validate validates a Properties struct
func (p *Properties) Validate() error {
	switch p.ProvisioningState {
	case "", Creating, Updating, Failed, Succeeded, Deleting, Migrating, Upgrading:
	default:
		return fmt.Errorf("invalid provisioningState %q", p.ProvisioningState)
	}

	if p.OpenShiftVersion != "3.10" {
		return fmt.Errorf("invalid openShiftVersion %q", p.OpenShiftVersion)
	}

	if !isValidHostname(p.PublicHostname) {
		return fmt.Errorf("invalid publicHostname %q", p.PublicHostname)
	}

	if !isValidHostname(p.RoutingConfigSubdomain) {
		return fmt.Errorf("invalid routingConfigSubdomain %q", p.RoutingConfigSubdomain)
	}

	return p.AgentPoolProfiles.Validate()
}

// Validate validates an AgentPoolProfiles slice
func (apps AgentPoolProfiles) Validate() error {
	names := map[string]struct{}{}

	for i := 1; i < len(apps); i++ {
		if apps[i].VnetSubnetID != apps[i-1].VnetSubnetID {
			return fmt.Errorf("non-identical vnetSubnetIDs")
		}
	}

	for _, app := range apps {
		if _, found := names[app.Name]; found {
			return fmt.Errorf("duplicate name %q", app.Name)
		}
		names[app.Name] = struct{}{}

		if err := app.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates an AgentPoolProfile struct
func (app *AgentPoolProfile) Validate() error {
	if !regexAgentPoolName.MatchString(app.Name) {
		return fmt.Errorf("invalid name %q", app.Name)
	}

	if app.Count < 0 || app.Count > 100 {
		return fmt.Errorf("invalid count %q", app.Count)
	}

	switch app.VMSize {
	case "Standard_D2s_v3", "Standard_D4s_v3":
	default:
		return fmt.Errorf("invalid vmSize %q", app.VMSize)
	}

	return nil
}
