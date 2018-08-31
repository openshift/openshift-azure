package api

// ContextKey is a type for context property bag payload keys
type ContextKey string

const (
	ContextKeyClientID       ContextKey = "ClientID"
	ContextKeyClientSecret   ContextKey = "ClientSecret"
	ContextKeyTenantID       ContextKey = "TenantID"
	ContextKeySubscriptionId ContextKey = "SubscriptionId"
	ContextKeyResourceGroup  ContextKey = "ResourceGroup"
)

// TypeMeta describes an individual API model object
type TypeMeta struct {
	// APIVersion is on every object
	APIVersion string `json:"apiVersion"`
}

// ResourcePurchasePlan defines resource plan as required by ARM
// for billing purposes.
type ResourcePurchasePlan struct {
	Name          string `json:"name"`
	Product       string `json:"product"`
	PromotionCode string `json:"promotionCode"`
	Publisher     string `json:"publisher"`
}

// ContainerService complies with the ARM model of
// resource definition in a JSON template.
type OpenShiftManagedCluster struct {
	ID       string                `json:"id"`
	Location string                `json:"location"`
	Name     string                `json:"name"`
	Plan     *ResourcePurchasePlan `json:"plan,omitempty"`
	Tags     map[string]string     `json:"tags"`
	Type     string                `json:"type"`

	Properties *Properties `json:"properties,omitempty"`
	Config     *Config     `json:"config,omitempty"`
}

// Properties represents the ACS cluster definition
type Properties struct {
	ProvisioningState       ProvisioningState        `json:"provisioningState,omitempty"`
	OrchestratorProfile     *OrchestratorProfile     `json:"orchestratorProfile,omitempty"`
	AgentPoolProfiles       []*AgentPoolProfile      `json:"agentPoolProfiles,omitempty"`
	ServicePrincipalProfile *ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`
	AzProfile               *AzProfile               `json:"azProfile,omitempty"`
	AuthProfile             *AuthProfile             `json:"authProfile,omitempty"`
	// Master LB public endpoint/FQDN with port
	// The format will be FQDN:2376
	// Not used during PUT, returned as part of GET
	FQDN string `json:"fqdn,omitempty"`
}

// AzProfile holds the azure context for where the cluster resides
type AzProfile struct {
	TenantID       string `json:"tenantId,omitempty"`
	SubscriptionID string `json:"subscriptionId,omitempty"`
	ResourceGroup  string `json:"resourceGroup,omitempty"`
}

// ServicePrincipalProfile contains the client and secret used by the cluster for Azure Resource CRUD
type ServicePrincipalProfile struct {
	ClientID string `json:"clientId"`
	Secret   string `json:"secret,omitempty" conform:"redact"`
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
	// Succeeded means resource created succeeded during last create/update
	Succeeded ProvisioningState = "Succeeded"
	// Deleting means resource is in the process of being deleted
	Deleting ProvisioningState = "Deleting"
	// Migrating means resource is being migrated from one subscription or
	// resource group to another
	Migrating ProvisioningState = "Migrating"
	// Upgrading means an existing ContainerService resource is being upgraded
	Upgrading ProvisioningState = "Upgrading"
)

// OrchestratorProfile contains Orchestrator properties
type OrchestratorProfile struct {
	OrchestratorVersion string           `json:"orchestratorVersion"`
	OpenShiftConfig     *OpenShiftConfig `json:"openshiftConfig,omitempty"`
}

// OpenShiftConfig holds configuration for OpenShift
type OpenShiftConfig struct {
	PublicHostname string
	RouterProfiles []OpenShiftRouterProfile
}

// OpenShiftRouterProfile represents an OpenShift router.
type OpenShiftRouterProfile struct {
	Name            string
	PublicSubdomain string
	FQDN            string
}

// AgentPoolProfile represents an agent pool definition
type AgentPoolProfile struct {
	Name         string               `json:"name"`
	Count        int                  `json:"count"`
	VMSize       string               `json:"vmSize"`
	DNSPrefix    string               `json:"dnsPrefix,omitempty"`
	OSType       OSType               `json:"osType,omitempty"`
	Ports        []int                `json:"ports,omitempty"`
	VnetSubnetID string               `json:"vnetSubnetID,omitempty"`
	Role         AgentPoolProfileRole `json:"role,omitempty"`
}

// AuthProfile defines all possible authentication profiles for OpenShift cluster
type AuthProfile struct {
	IdentityProviders []IdentityProvider `json:"identityProviders,omitempty"`
}

// IdentityProvider is heavily cut down equivalent to IdentityProvider in the upstream
type IdentityProvider struct {
	Name     string      `json:"name"`
	Provider interface{} `json:"provider,omityempty"`
}

// AADIdentityProvider defines Identity provider for MS AAD
// it is based on OpenID IdentityProvider
type AADIdentityProvider struct {
	Kind     string `json:"kind,omitempty"`
	ClientID string `json:"clientId,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

// AgentPoolProfileRole represents an agent role
type AgentPoolProfileRole string

const (
	// AgentPoolProfileRoleCompute is the compute role.
	AgentPoolProfileRoleCompute AgentPoolProfileRole = "compute"
	// AgentPoolProfileRoleInfra is the infra role.
	AgentPoolProfileRoleInfra AgentPoolProfileRole = "infra"
	// AgentPoolProfileRoleMaster is the master role.
	AgentPoolProfileRoleMaster AgentPoolProfileRole = "master"
)

// OSType represents OS types of agents
type OSType string

const (
	// OSTypeLinux is Linux.
	OSTypeLinux OSType = "Linux"
	// OSTypeWindows is Windows.
	OSTypeWindows OSType = "Windows"
)

// Distro represents Linux distro to use for Linux VMs
type Distro string

// TotalNodes returns the total number of nodes in the cluster configuration
func (p *Properties) TotalNodes() int {
	var totalNodes int
	for _, pool := range p.AgentPoolProfiles {
		totalNodes = totalNodes + pool.Count
	}
	return totalNodes
}

// IsCustomVNET returns true if the customer brought their own VNET
func (a *AgentPoolProfile) IsCustomVNET() bool {
	return len(a.VnetSubnetID) > 0
}
