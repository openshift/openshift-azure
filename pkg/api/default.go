package api

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
