package config

import (
	"strings"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

// ReturnDerivedKubeArguments takes in cluster configuration, agent role and required kube-arguments and returns value
func ReturnDerivedKubeArguments(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole, argument acsapi.KubeletArguments) string {
	switch argument {
	case acsapi.KubeletArgumentsKubeReserved:
		return derivedKubeReserved(cs, role)
	case acsapi.KubeletArgumentsSystemReserved:
		return derivedSystemReserved(cs, role)
	default:
		panic("ReturnDerivedKubeArguments failed to derive values")
	}
}

func derivedSystemReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}

		switch pool.VMSize {
		//TODO: Move VMs sized to its own type in all code-base
		case "Standard_D2s_v3":
			return "cpu=200m,memory=512Mi"
		case "Standard_D4s_v3":
			return "cpu=500m,memory=512Mi"
		default:
			panic("derivedSystemReserved failed to derive values")
		}
	}
	return ""
}

func derivedKubeReserved(cs *acsapi.OpenShiftManagedCluster, role acsapi.AgentPoolProfileRole) string {
	for _, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Role != role {
			continue
		}

		switch pool.VMSize {
		case "Standard_D2s_v3":
			return "cpu=200m,memory=512Mi"
		case "Standard_D4s_v3":
			return "cpu=500m,memory=512Mi"
		default:
			panic("derivedKubeReserved failed to derive values")
		}
	}
	return ""
}

func ReturnDerivedRouterLBCNamePrefix(cs *acsapi.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN, ".")[0]
}
