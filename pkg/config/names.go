package config

import (
	"fmt"
)

// GetScalesetName returns the VMSS name for a given AgentPoolProfile
func GetScalesetName(appName string) string {
	return "ss-" + appName
}

// GetInstanceName returns the VMSS instance name for a given AgentPoolProfile
// name and instance number
func GetInstanceName(appName string, instance int) string {
	return fmt.Sprintf("ss-%s_%d", appName, instance)
}
