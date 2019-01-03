package config

import (
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
)

const MasterScalesetName = "ss-master"

// GetScalesetName returns the VMSS name for a given AgentPoolProfile
func GetScalesetName(app *api.AgentPoolProfile, suffix string) string {
	if app.Role == api.AgentPoolProfileRoleMaster {
		return MasterScalesetName
	}
	return "ss-" + app.Name + "-" + suffix
}

// GetComputerNamePrefix returns the computer name prefix for a given
// AgentPoolProfile
func GetComputerNamePrefix(app *api.AgentPoolProfile, suffix string) string {
	if app.Role == api.AgentPoolProfileRoleMaster {
		return app.Name + "-"
	}
	return app.Name + "-" + suffix + "-"
}

// GetMasterInstanceName returns the VMSS instance name for a given
// AgentPoolProfile name and instance number
func GetMasterInstanceName(instance int64) string {
	return MasterScalesetName + fmt.Sprintf("_%d", instance)
}
