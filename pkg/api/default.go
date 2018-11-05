package api

// TODO: calculate the reserved resource sizes based on instance resources?
func getAgentPoolProfile(role AgentPoolProfileRole, small bool) map[ReservedResource]string {
	// default resources
	var baseline, master string
	if small {
		baseline = "cpu=200m,memory=512Mi"
		master = "cpu=500m,memory=1Gi"
	} else {
		baseline = "cpu=500m,memory=512Mi"
		master = "cpu=1000m,memory=1Gi"
	}
	var reservedResources = map[ReservedResource]string{
		SystemReserved: baseline,
	}
	if role == AgentPoolProfileRoleMaster {
		reservedResources[SystemReserved] = master
	} else {
		// we only set the kube reserved resources for non-master roles
		reservedResources[KubeReserved] = baseline
	}
	return reservedResources
}

func getAgentPoolProfiles(size VMSize) map[AgentPoolProfileRole]map[ReservedResource]string {
	// The smallest sizes are used for QA, not intended for production usage
	var small = size == StandardD2sV3 || size == StandardE2sV3
	var agentPoolProfiles = map[AgentPoolProfileRole]map[ReservedResource]string{
		AgentPoolProfileRoleMaster:  getAgentPoolProfile(AgentPoolProfileRoleMaster, small),
		AgentPoolProfileRoleInfra:   getAgentPoolProfile(AgentPoolProfileRoleInfra, small),
		AgentPoolProfileRoleCompute: getAgentPoolProfile(AgentPoolProfileRoleCompute, small),
	}
	return agentPoolProfiles
}

// DefaultVMSizeKubeArguments defines default values of kube-arguments based on the VM size
var DefaultVMSizeKubeArguments = map[VMSize]map[AgentPoolProfileRole]map[ReservedResource]string{
	// standard instances
	StandardD2sV3:  getAgentPoolProfiles(StandardD2sV3),
	StandardD4sV3:  getAgentPoolProfiles(StandardD4sV3),
	StandardD8sV3:  getAgentPoolProfiles(StandardD8sV3),
	StandardD16sV3: getAgentPoolProfiles(StandardD16sV3),
	StandardD32sV3: getAgentPoolProfiles(StandardD32sV3),
	StandardD64sV3: getAgentPoolProfiles(StandardD64sV3),

	// memory optimized instances
	StandardE2sV3:  getAgentPoolProfiles(StandardE2sV3),
	StandardE4sV3:  getAgentPoolProfiles(StandardE4sV3),
	StandardE8sV3:  getAgentPoolProfiles(StandardE8sV3),
	StandardE16sV3: getAgentPoolProfiles(StandardE16sV3),
	StandardE20sV3: getAgentPoolProfiles(StandardE20sV3),
	StandardE32sV3: getAgentPoolProfiles(StandardE32sV3),
	StandardE64sV3: getAgentPoolProfiles(StandardE64sV3),

	// storage optimized instances
	StandardL4s:  getAgentPoolProfiles(StandardL4s),
	StandardL8s:  getAgentPoolProfiles(StandardL8s),
	StandardL16s: getAgentPoolProfiles(StandardL16s),
	StandardL32s: getAgentPoolProfiles(StandardL32s),
	StandardL64s: getAgentPoolProfiles(StandardL64s),
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
