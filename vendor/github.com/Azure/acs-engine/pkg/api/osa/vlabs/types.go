package vlabs

// OpenShiftCluster complies with the ARM model of resource definition in a JSON
// template.
type OpenShiftCluster struct {
	ID         string                `json:"id,omitempty"`
	Location   string                `json:"location,omitempty"`
	Name       string                `json:"name,omitempty"`
	Plan       *ResourcePurchasePlan `json:"plan,omitempty"`
	Tags       map[string]string     `json:"tags,omitempty"`
	Type       string                `json:"type,omitempty"`
	Properties Properties            `json:"properties,omitempty"`
}

// ResourcePurchasePlan defines the resource plan as required by ARM for billing
// purposes.
type ResourcePurchasePlan struct {
	Name          string `json:"name,omitempty"`
	Product       string `json:"product,omitempty"`
	PromotionCode string `json:"promotionCode,omitempty"`
	Publisher     string `json:"publisher,omitempty"`
}

// Properties represents the cluster definition.
type Properties struct {
	ProvisioningState       ProvisioningState       `json:"provisioningState,omitempty"`
	OpenShiftVersion        string                  `json:"openShiftVersion,omitempty"`
	PublicHostname          string                  `json:"publicHostname,omitempty"`
	RoutingConfigSubdomain  string                  `json:"routingConfigSubdomain,omitempty"`
	AgentPoolProfiles       AgentPoolProfiles       `json:"agentPoolProfiles,omitempty"`
	ServicePrincipalProfile ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`
}

// ProvisioningState represents the current state of container service resource.
type ProvisioningState string

const (
	// Creating means ContainerService resource is being created.
	Creating ProvisioningState = "Creating"
	// Updating means an existing ContainerService resource is being updated
	Updating ProvisioningState = "Updating"
	// Failed means resource is in failed state
	Failed ProvisioningState = "Failed"
	// Succeeded means last create/update succeeded
	Succeeded ProvisioningState = "Succeeded"
	// Deleting means resource is in the process of being deleted
	Deleting ProvisioningState = "Deleting"
	// Migrating means resource is being migrated from one subscription or
	// resource group to another
	Migrating ProvisioningState = "Migrating"
	// Upgrading means an existing resource is being upgraded
	Upgrading ProvisioningState = "Upgrading"
)

// AgentPoolProfiles represents all the AgentPoolProfiles
type AgentPoolProfiles []AgentPoolProfile

// AgentPoolProfile represents configuration of VMs running agent daemons that
// register with the master and offer resources to host applications in
// containers.
type AgentPoolProfile struct {
	Name         string `json:"name,omitempty"`
	Count        int    `json:"count,omitempty"`
	VMSize       string `json:"vmSize,omitempty"`
	VnetSubnetID string `json:"vnetSubnetID,omitempty"`
}

// ServicePrincipalProfile contains the client and secret used by the cluster
// for Azure Resource CRUD.
type ServicePrincipalProfile struct {
	ClientID string `json:"clientId,omitempty"`
	Secret   string `json:"secret,omitempty"`
}
