package config

import (
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

func DerivedSystemReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}
		return acsapi.DefaultVMSizeKubeArguments[pool.VMSize]["system-reserved"]
	}
	return ""
}

func DerivedKubeReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}
		return acsapi.DefaultVMSizeKubeArguments[pool.VMSize]["kube-reserved"]
	}
	return ""
}

func DerivedRouterLBCNamePrefix(cs *acsapi.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.RouterProfiles[0].FQDN, ".")[0]
}
