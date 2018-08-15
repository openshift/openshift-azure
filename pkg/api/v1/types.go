package v1

// TODO publish v1 to a dated api as in Azure conventions

// OpenShiftManagedCluster complies with the ARM model of resource definition in a JSON
// template.
type OpenShiftManagedCluster struct {
	ID         string                `json:"id,omitempty"`
	Location   string                `json:"location,omitempty"`
	Name       string                `json:"name,omitempty"`
	Plan       *ResourcePurchasePlan `json:"plan,omitempty"`
	Tags       map[string]string     `json:"tags,omitempty"`
	Type       string                `json:"type,omitempty"`
	Properties *Properties           `json:"properties,omitempty"`
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
	// ProvisioningState (out): current state of the OSA resource.
	ProvisioningState ProvisioningState `json:"provisioningState,omitempty"`

	// OpenShiftVersion (in): OpenShift version to be created/updated, e.g.
	// `v3.10`.
	OpenShiftVersion string `json:"openShiftVersion,omitempty"`

	// PublicHostname (in,optional): Optional user-specified FQDN for OpenShift
	// API server.  If specified, after OSA cluster creation, user must create a
	// PublicHostname CNAME record forwarding to the returned FQDN value.
	PublicHostname string `json:"publicHostname,omitempty"`

	// FQDN (out): Auto-allocated FQDN for OpenShift API server.
	FQDN string `json:"fqdn,omitempty"`

	// RouterProfiles (in,optional/out): Configuration for OpenShift router(s).
	RouterProfiles []RouterProfile `json:"routerProfiles,omitempty"`

	// MasterPoolProfile (in): Configuration for OpenShift master VMs.
	MasterPoolProfile MasterPoolProfile `json:"masterPoolProfile,omitempty"`

	// AgentPoolProfiles (in): configuration of OpenShift cluster VMs.
	AgentPoolProfiles []AgentPoolProfile `json:"agentPoolProfiles,omitempty"`

	// TODO: is this compatible with MSI?
	// ServicePrincipalProfile (in): Service principal for OpenShift cluster.
	ServicePrincipalProfile ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`
}

// ProvisioningState represents the current state of the OSA resource.
type ProvisioningState string

const (
	// Creating means the OSA resource is being created.
	Creating ProvisioningState = "Creating"
	// Updating means the existing OSA resource is being updated.
	Updating ProvisioningState = "Updating"
	// Failed means the OSA resource is in failed state.
	Failed ProvisioningState = "Failed"
	// Succeeded means the last create/update succeeded.
	Succeeded ProvisioningState = "Succeeded"
	// Deleting means the OSA resource is being deleted.
	Deleting ProvisioningState = "Deleting"
	// Migrating means the OSA resource is being migrated from one subscription
	// or resource group to another.
	Migrating ProvisioningState = "Migrating"
	// Upgrading means the existing OAS resource is being upgraded.
	Upgrading ProvisioningState = "Upgrading"
)

// RouterProfile represents an OpenShift router.
type RouterProfile struct {
	Name string `json:"name,omitempty"`

	// PublicSubdomain (in,optional/out): DNS subdomain for OpenShift router. If
	// specified, after OSA cluster creation, user must create a (wildcard)
	// *.PublicSubdomain CNAME record forwarding to the returned FQDN value.  If
	// not specified, OSA will auto-allocate and setup a PublicSubdomain and
	// return it.  The OpenShift master is configured with the PublicSubdomain
	// of the "default" RouterProfile.
	PublicSubdomain string `json:"publicSubdomain,omitempty"`

	// FQDN (out): Auto-allocated FQDN for the OpenShift router.
	FQDN string `json:"fqdn,omitempty"`
}

// MasterPoolProfile contains configuration for OpenShift master VMs.
type MasterPoolProfile struct {
	ProfileSpec
}

// AgentPoolProfile represents configuration of OpenShift cluster VMs.
type AgentPoolProfile struct {
	ProfileSpec
	Role AgentPoolProfileRole `json:"role,omitempty"`
}

// ProfileSpec contains all shared fields that are used across MasterProfile
// and AgentPoolProfile.  Items should only be added here if they are shared.
type ProfileSpec struct {
	Name   string `json:"name,omitempty"`
	Count  int    `json:"count,omitempty"`
	VMSize string `json:"vmSize,omitempty"`

	// VnetSubnetID is expected to be empty or match
	// `^/subscriptions/[^/]+
	//   /resourceGroups/[^/]+
	//   /providers/Microsoft.Network
	//   /virtualNetworks/[^/]+
	//   /subnets/[^/]+$`
	VnetSubnetID string `json:"vnetSubnetID,omitempty"`
	OSType       OSType `json:"osType,omitempty"`
}

// AgentPoolProfileRole represents the role of the AgentPoolProfile.
type AgentPoolProfileRole string

const (
	// AgentPoolProfileRoleCompute is the compute role.
	AgentPoolProfileRoleCompute AgentPoolProfileRole = "compute"
	// AgentPoolProfileRoleInfra is the infra role.
	AgentPoolProfileRoleInfra AgentPoolProfileRole = "infra"
)

// OSType represents the OS type of VMs in an AgentPool.
type OSType string

const (
	// OSTypeLinux is Linux.
	OSTypeLinux OSType = "Linux"
	// OSTypeWindows is Windows.
	OSTypeWindows OSType = "Windows"
)

// ServicePrincipalProfile contains the client and secret used by the cluster
// for Azure Resource CRUD.
type ServicePrincipalProfile struct {
	ClientID string `json:"clientId,omitempty"`
	Secret   string `json:"secret,omitempty"`
}
