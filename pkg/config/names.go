package config

import (
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
)

// GetScalesetName get the scaleset name as built in the arm template
func GetScalesetName(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) string {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			return "ss-" + app.Name
		}
	}
	panic("invalid role")
}

// GetInstanceName returns the VMSS instance name for a given AgentPoolProfile
// name and instance number
func GetInstanceName(appName string, instance int) string {
	return fmt.Sprintf("ss-%s_%d", appName, instance)
}
