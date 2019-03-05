package api

const (
	// APIVersion is the version of this API
	APIVersion = "internal"
)

// OpenShiftManagedCluster complies with the ARM model of resource definition in
// a JSON template.
type OpenShiftManagedCluster struct {
	Plan       *ResourcePurchasePlan `json:"plan,omitempty"`
	Properties Properties            `json:"properties,omitempty"`
	ID         string                `json:"id,omitempty"`
	Name       string                `json:"name,omitempty"`
	Type       string                `json:"type,omitempty"`
	Location   string                `json:"location,omitempty"`
	Tags       map[string]string     `json:"tags"`

	Config Config `json:"config,omitempty"`
}

// ResourcePurchasePlan defines the resource plan as required by ARM for billing
// purposes.
type ResourcePurchasePlan struct {
	Name          *string `json:"name,omitempty"`
	Product       *string `json:"product,omitempty"`
	PromotionCode *string `json:"promotionCode,omitempty"`
	Publisher     *string `json:"publisher,omitempty"`
}

// Properties represents the cluster definition.
type Properties struct {
	// ProvisioningState (out): current state of the OSA resource.
	ProvisioningState ProvisioningState `json:"provisioningState,omitempty"`

	// OpenShiftVersion (in): OpenShift version to be created/updated, e.g.
	// `v3.11`.
	OpenShiftVersion string `json:"openShiftVersion,omitempty"`

	// PublicHostname (in,optional): Optional user-specified FQDN for OpenShift
	// API server.  If specified, after OSA cluster creation, user must create a
	// PublicHostname CNAME record forwarding to the returned FQDN value.
	PublicHostname string `json:"publicHostname,omitempty"`

	// FQDN (in): FQDN for OpenShift API server.  User-specified FQDN for
	// OpenShift API server loadbalancer internal hostname.
	FQDN string `json:"fqdn,omitempty"`

	// NetworkProfile (in): Configuration for OpenShift networking.
	NetworkProfile NetworkProfile `json:"networkProfile,omitempty"`

	// RouterProfiles (in,optional/out): Configuration for OpenShift router(s).
	RouterProfiles []RouterProfile `json:"routerProfiles,omitempty"`

	// AgentPoolProfiles (in): configuration of OpenShift cluster VMs.
	AgentPoolProfiles []AgentPoolProfile `json:"agentPoolProfiles,omitempty"`

	// AuthProfile (in): configures OpenShift authentication
	AuthProfile AuthProfile `json:"authProfile,omitempty"`

	// MasterServicePrincipalProfile service principle used on the masters
	MasterServicePrincipalProfile ServicePrincipalProfile `json:"masterServicePrincipalProfile,omitempty"`

	// WorkerServicePrincipalProfile service principle used on the nodes
	WorkerServicePrincipalProfile ServicePrincipalProfile `json:"workerServicePrincipalProfile,omitempty"`

	AzProfile AzProfile `json:"azProfile,omitempty"`

	// APICertProfile (in, optional): configures OpenShift API certificate
	APICertProfile CertProfile `json:"apiCertProfile,omitempty"`
}

// ProvisioningState represents the current state of the OSA resource.
type ProvisioningState string

const (
	// Creating means the OSA resource is being created.
	Creating ProvisioningState = "Creating"
	// Updating means the existing OSA resource is being updated.
	Updating ProvisioningState = "Updating"
	// AdminUpdating means the existing OSA resource is being updated with admin privileges.
	AdminUpdating ProvisioningState = "AdminUpdating"
	// Failed means the OSA resource is in failed state.
	Failed ProvisioningState = "Failed"
	// Succeeded means the last create/update succeeded.
	Succeeded ProvisioningState = "Succeeded"
	// Deleting means the OSA resource is being deleted.
	Deleting ProvisioningState = "Deleting"
	// Migrating means the OSA resource is being migrated from one subscription
	// or resource group to another.
	Migrating ProvisioningState = "Migrating"
	// Upgrading means the existing OSA resource is being upgraded.
	Upgrading ProvisioningState = "Upgrading"
)

// NetworkProfile contains configuration for OpenShift networking.
type NetworkProfile struct {
	// VnetCIDR (in): the CIDR with which the OSA cluster's Vnet is configured
	VnetCIDR string `json:"vnetCidr,omitempty"`

	// VnetID (out): the ID of the Vnet created for the OSA cluster
	VnetID string `json:"vnetId,omitempty"`

	// PeerVnetID (in, optional): ID of a Vnet with which the OSA cluster Vnet should be peered.
	// If specified, this should match
	// `^/subscriptions/[^/]+
	//   /resourceGroups/[^/]+
	//   /providers/Microsoft.Network
	//   /virtualNetworks/[^/]+$`
	PeerVnetID *string `json:"peerVnetId,omitempty"`
}

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

	// RouterCertProfile (in, optional): configures OpenShift Router certificate
	RouterCertProfile CertProfile `json:"routerCertProfile,omitempty"`
}

// CertProfile contains the vault location for OpenShift certificates.
type CertProfile struct {
	// KeyVaultSecretURL (in, optional): the customer secret URL
	KeyVaultSecretURL string `json:"keyVaultSecretURL,omitempty"`
}

// AgentPoolProfile represents configuration of OpenShift cluster VMs.
type AgentPoolProfile struct {
	Name       string `json:"name,omitempty"`
	Count      int64  `json:"count,omitempty"`
	VMSize     VMSize `json:"vmSize,omitempty"`
	SubnetCIDR string `json:"subnetCidr,omitempty"`
	OSType     OSType `json:"osType,omitempty"`

	Role AgentPoolProfileRole `json:"role,omitempty"`
}

// OSType represents the OS type of VMs in an AgentPool.
type OSType string

const (
	// OSTypeLinux is Linux.
	OSTypeLinux OSType = "Linux"
	// OSTypeWindows is Windows.
	OSTypeWindows OSType = "Windows"
)

// AgentPoolProfileRole represents the role of the AgentPoolProfile.
type AgentPoolProfileRole string

const (
	// AgentPoolProfileRoleCompute is the compute role.
	AgentPoolProfileRoleCompute AgentPoolProfileRole = "compute"
	// AgentPoolProfileRoleInfra is the infra role.
	AgentPoolProfileRoleInfra AgentPoolProfileRole = "infra"
	// AgentPoolProfileRoleMaster is the master role.
	AgentPoolProfileRoleMaster AgentPoolProfileRole = "master"
)

// VMSize represents supported VMSizes
type VMSize string

// VMSizes.  Keep in sync with MaxDataDisksPerVM()
const (
	// General purpose VMs
	StandardD2sV3  VMSize = "Standard_D2s_v3"
	StandardD4sV3  VMSize = "Standard_D4s_v3"
	StandardD8sV3  VMSize = "Standard_D8s_v3"
	StandardD16sV3 VMSize = "Standard_D16s_v3"
	StandardD32sV3 VMSize = "Standard_D32s_v3"

	// Compute optimized VMs
	StandardF8sV2  VMSize = "Standard_F8s_v2"
	StandardF16sV2 VMSize = "Standard_F16s_v2"
	StandardF32sV2 VMSize = "Standard_F32s_v2"

	// Memory optimized VMs
	StandardE4sV3  VMSize = "Standard_E4s_v3"
	StandardE8sV3  VMSize = "Standard_E8s_v3"
	StandardE16sV3 VMSize = "Standard_E16s_v3"
	StandardE20sV3 VMSize = "Standard_E20s_v3"
	StandardE32sV3 VMSize = "Standard_E32s_v3"

	// Storage optimized VMs
	StandardL4s  VMSize = "Standard_L4s"
	StandardL8s  VMSize = "Standard_L8s"
	StandardL16s VMSize = "Standard_L16s"
	StandardL32s VMSize = "Standard_L32s"
)

// AuthProfile defines all possible authentication profiles for the OpenShift
// cluster.
type AuthProfile struct {
	IdentityProviders []IdentityProvider `json:"identityProviders,omitempty"`
}

// IdentityProvider is heavily cut down equivalent to IdentityProvider in the
// upstream.
type IdentityProvider struct {
	Name     string      `json:"name,omitempty"`
	Provider interface{} `json:"provider,omitempty"`
}

// AADIdentityProvider defines Identity provider for MS AAD.  It is based on
// OpenID IdentityProvider.
type AADIdentityProvider struct {
	Kind     string `json:"kind,omitempty"`
	ClientID string `json:"clientId,omitempty"`
	Secret   string `json:"secret,omitempty"`
	TenantID string `json:"tenantId,omitempty"`
	// CustomerAdminGroupID group memberships will get synced into the OpenShift group "osa-customer-admins"
	CustomerAdminGroupID *string `json:"customerAdminGroupId,omitempty"`
}

// ServicePrincipalProfile contains the client and secret used by the cluster
// for Azure Resource CRUD.
type ServicePrincipalProfile struct {
	ClientID string `json:"clientId,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

// AzProfile holds the azure context for where the cluster resides
type AzProfile struct {
	TenantID       string `json:"tenantId,omitempty"`
	SubscriptionID string `json:"subscriptionId,omitempty"`
	ResourceGroup  string `json:"resourceGroup,omitempty"`
}
