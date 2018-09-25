package api

// DefaultVMSizeKubeArguments defines default values of kube-arguments based on the VM size
var DefaultVMSizeKubeArguments = map[VMSize]map[AgentPoolProfileRole]map[ReservedResource]string{
	StandardD2sV3: {
		AgentPoolProfileRoleMaster: {
			SystemReserved: "cpu=500m,memory=1Gi",
		},
		AgentPoolProfileRoleCompute: {
			KubeReserved:   "cpu=200m,memory=512Mi",
			SystemReserved: "cpu=200m,memory=512Mi",
		},
		AgentPoolProfileRoleInfra: {
			KubeReserved:   "cpu=200m,memory=512Mi",
			SystemReserved: "cpu=200m,memory=512Mi",
		},
	},
	StandardD4sV3: {
		AgentPoolProfileRoleMaster: {
			SystemReserved: "cpu=1000m,memory=1Gi",
		},
		AgentPoolProfileRoleCompute: {
			KubeReserved:   "cpu=500m,memory=512Mi",
			SystemReserved: "cpu=500m,memory=512Mi",
		},
		AgentPoolProfileRoleInfra: {
			KubeReserved:   "cpu=500m,memory=512Mi",
			SystemReserved: "cpu=500m,memory=512Mi",
		},
	},
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
