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

// GetSecurityGroupName get the name of the security group
func GetSecurityGroupName(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) string {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			return "nsg-" + app.Name
		}
	}
	panic("invalid role")
}

// GetInstanceName get the instance name as built in the arm template
func GetInstanceName(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole, instance int) string {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			return fmt.Sprintf("ss-%s_%d", app.Name, instance)
		}
	}
	panic("invalid role")
}
