package api

import "strings"

// TODO: calculate the reserved resource sizes based on instance resources
func getAgentPoolProfile(role string) map[ReservedResource]string {
	// default resources
	var reservedResources = map[ReservedResource]string{
		SystemReserved: "cpu=200m,memory=512Mi",
	}
	if strings.ToLower(role) == "master" {
		// bump up master's system reserves
		reservedResources[SystemReserved] = "cpu=500m,memory=1Gi"
	} else {
		// we only set the kube reserved resources for non-master roles
		reservedResources[KubeReserved] = "cpu=200m,memory=512Mi"
	}
	return reservedResources
}

func getAgentPoolProfiles() map[AgentPoolProfileRole]map[ReservedResource]string {
	var agentPoolProfiles = map[AgentPoolProfileRole]map[ReservedResource]string{
		AgentPoolProfileRoleMaster:  getAgentPoolProfile("master"),
		AgentPoolProfileRoleInfra:   getAgentPoolProfile("infra"),
		AgentPoolProfileRoleCompute: getAgentPoolProfile("compute"),
	}
	return agentPoolProfiles
}

// DefaultVMSizeKubeArguments defines default values of kube-arguments based on the VM size
var DefaultVMSizeKubeArguments = map[VMSize]map[AgentPoolProfileRole]map[ReservedResource]string{
	// standard instances
	StandardD2sV3:  getAgentPoolProfiles(),
	StandardD4sV3:  getAgentPoolProfiles(),
	StandardD8sV3:  getAgentPoolProfiles(),
	StandardD16sV3: getAgentPoolProfiles(),
	StandardD32sV3: getAgentPoolProfiles(),
	StandardD64sV3: getAgentPoolProfiles(),

	// memory optimized instances
	StandardE2sV3:  getAgentPoolProfiles(),
	StandardE4sV3:  getAgentPoolProfiles(),
	StandardE8sV3:  getAgentPoolProfiles(),
	StandardE16sV3: getAgentPoolProfiles(),
	StandardE20sV3: getAgentPoolProfiles(),
	StandardE32sV3: getAgentPoolProfiles(),
	StandardE64sV3: getAgentPoolProfiles(),

	// storage optimized instances
	StandardL4s:  getAgentPoolProfiles(),
	StandardL8s:  getAgentPoolProfiles(),
	StandardL16s: getAgentPoolProfiles(),
	StandardL32s: getAgentPoolProfiles(),
	StandardL64s: getAgentPoolProfiles(),
}

// AzureLocations defines a) known regions where we permit OSA to be deployed,
// and b) mapping from OSA region to a LogAnalytics region.  Logging data sent
// to LogAnalytics must remain in the same sovereign.  See
// https://azure.microsoft.com/en-us/global-infrastructure/geographies/ .
var AzureLocations = map[string]string{
	"australiacentral":   "australiasoutheast",
	"australiacentral2":  "australiasoutheast",
	"australiaeast":      "australiasoutheast",
	"australiasoutheast": "australiasoutheast",
	"canadacentral":      "canadacentral",
	"canadaeast":         "canadacentral",
	"centralindia":       "centralindia",
	"centralus":          "eastus",
	"eastasia":           "southeastasia",
	"eastus":             "eastus",
	"eastus2":            "eastus",
	"japaneast":          "japaneast",
	"japanwest":          "japaneast",
	"northcentralus":     "eastus",
	"northeurope":        "westeurope",
	"southcentralus":     "eastus",
	"southeastasia":      "southeastasia",
	"southindia":         "centralindia",
	"uksouth":            "uksouth",
	"ukwest":             "uksouth",
	"westcentralus":      "eastus",
	"westeurope":         "westeurope",
	"westindia":          "centralindia",
	"westus":             "eastus",
	"westus2":            "eastus",
}
