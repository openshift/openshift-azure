package validate

import (
	"fmt"
	"os"

	acsapi "github.com/Azure/acs-engine/pkg/api"
)

func Validate(cs, _ *acsapi.ContainerService) error {
	if cs.Properties == nil ||
		cs.Properties.OrchestratorProfile == nil ||
		cs.Properties.OrchestratorProfile.OpenShiftConfig == nil ||
		cs.Properties.ServicePrincipalProfile == nil {
		return fmt.Errorf("malformed manifest")
	}

	switch cs.Properties.OrchestratorProfile.OpenShiftConfig.OpenShiftVersion {
	case "v3.10":
	default:
		return fmt.Errorf("invalid openShiftVersion %q", cs.Properties.OrchestratorProfile.OpenShiftConfig.OpenShiftVersion)
	}

	if len(cs.Properties.AgentPoolProfiles) != 3 {
		return fmt.Errorf("invalid number of agentPoolProfiles")
	}

	if cs.Properties.AgentPoolProfiles[0].VnetSubnetID != cs.Properties.AgentPoolProfiles[1].VnetSubnetID ||
		cs.Properties.AgentPoolProfiles[0].VnetSubnetID != cs.Properties.AgentPoolProfiles[2].VnetSubnetID {
		return fmt.Errorf("non-identical vnetSubnetIDs")
	}

	pools := map[string]*acsapi.AgentPoolProfile{}
	for _, app := range cs.Properties.AgentPoolProfiles {
		pools[app.Name] = app

		if app.Count > 100 {
			return fmt.Errorf("invalid count %q", app.Count)
		}

		switch app.VMSize {
		case "Standard_D2s_v3",
			"Standard_D4s_v3":
		default:
			return fmt.Errorf("invalid vmSize %q", app.VMSize)
		}
	}

	if pools["master"] == nil {
		return fmt.Errorf("missing master agentPoolProfile")
	}
	if pools["master"].Role != acsapi.AgentPoolProfileRoleMaster {
		return fmt.Errorf("invalid master agentPoolProfile role %q", pools["master"].Role)
	}
	if pools["master"].Count != 3 {
		return fmt.Errorf("invalid master agentPoolProfile count %q", pools["master"].Count)
	}
	if pools["infra"] == nil {
		return fmt.Errorf("missing infra agentPoolProfile")
	}
	if pools["infra"].Role != acsapi.AgentPoolProfileRoleInfra {
		return fmt.Errorf("invalid infra agentPoolProfile role %q", pools["infra"].Role)
	}
	if pools["infra"].Count == 0 {
		return fmt.Errorf("invalid infra agentPoolProfile count %q", pools["infra"].Count)
	}
	if pools["compute"] == nil {
		return fmt.Errorf("missing compute agentPoolProfile")
	}
	if pools["compute"].Role != acsapi.AgentPoolProfileRoleCompute {
		return fmt.Errorf("invalid compute agentPoolProfile role %q", pools["compute"].Role)
	}
	if pools["compute"].Count == 0 {
		return fmt.Errorf("invalid compute agentPoolProfile count %q", pools["compute"].Count)
	}

	return validateDevelopmentSwitches()
}

func validateDevelopmentSwitches() error {
	switch os.Getenv("DEPLOY_OS") {
	case "":
	case "centos7":
	default:
		return fmt.Errorf("invalid DEPLOY_OS %q", os.Getenv("DEPLOY_OS"))
	}

	return nil
}
