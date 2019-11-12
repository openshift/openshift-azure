package api

import (
	"time"
)

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

	// ClusterVersion (out): RP version at which cluster was last
	// created/updated
	ClusterVersion string `json:"clusterVersion,omitempty"`

	// PublicHostname (out): public hostname of OpenShift API server.
	PublicHostname string `json:"publicHostname,omitempty"`

	// FQDN (out): Auto-allocated internal FQDN for OpenShift API server.
	FQDN string `json:"fqdn,omitempty"`

	// PrivateAPIServer (internal only): Specifies if API server is public or private
	PrivateAPIServer bool `json:"privateApiServer,omitempty"`

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

	// MonitorProfile configures which log analytics workspace to use.
	MonitorProfile MonitorProfile `json:"monitorProfile,omitempty"`

	// APICertProfile (in, optional): configures OpenShift API certificate
	APICertProfile CertProfile `json:"apiCertProfile,omitempty"`

	// RefreshCluster (in, optional): if the nameservers are changed on
	// the vnet then allow the update to cause nodes to rotate.
	// Note:
	// 1. This is just a safety mechanism so this does not happen unintentionally
	//    but this *must* be done for the nodes to use of the new nameservers.
	// 2. This is a pointer and the public fields are not pointers. This is on
	//    purpose so we know if the request is coming from an older API.
	// 3. This is really a per request setting, not a cluster configuration.
	RefreshCluster *bool `json:"refreshCluster,omitempty"`
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

	// ManagementSubnetCIDR (in): the CIDR for OSA management subnet
	ManagementSubnetCIDR *string `json:"managementSubnetCidr,omitempty"`

	// VnetID (out): the ID of the Vnet created for the OSA cluster
	VnetID string `json:"vnetId,omitempty"`

	// PeerVnetID (in, optional): ID of a Vnet with which the OSA cluster Vnet should be peered.
	// If specified, this should match
	// `^/subscriptions/[^/]+
	//   /resourceGroups/[^/]+
	//   /providers/Microsoft.Network
	//   /virtualNetworks/[^/]+$`
	PeerVnetID *string `json:"peerVnetId,omitempty"`

	// PrivateEndpoint contains IP or DNS of PrivateEndpoint
	// RP sets the endpoint that the plugin should dial before
	// calling any plugin functions. This value is used based on
	// cluster type - Private/Public. Depending on cluster type
	// different endpoint will be used to reach API server -
	// PrivateEndpoint or FQDN.
	PrivateEndpoint *string `json:"-"`
	// ManagementSubnetID is ID of the management subnet.
	// Plugin ensures these are set to valid values
	// It be set during Generate plugin phase as they used by
	// the RP in the deployment callback function
	ManagementSubnetID string `json:"-"`
	// InternalLoadBalancerFrontendIPID is ID of the internal loadbalancer.
	// Plugin ensures these are set to valid values
	// It be set during Generate plugin phase as they used by
	// the RP in the deployment callback function
	InternalLoadBalancerFrontendIPID string `json:"-"`
	// Nameservers is the list of DNS nameservers that the running nodes
	// are using.
	Nameservers []string `json:"nameservers,omitempty"`
}

// RouterProfile represents an OpenShift router.
type RouterProfile struct {
	Name string `json:"name,omitempty"`

	// PublicSubdomain (out): DNS subdomain for OpenShift router.  The OpenShift
	// master is configured with the PublicSubdomain of the "default"
	// RouterProfile.
	PublicSubdomain string `json:"publicSubdomain,omitempty"`

	// FQDN (out): Auto-allocated internal FQDN for the OpenShift router.
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

// MonitorProfile configuration for Azure Monitor log analytics.
type MonitorProfile struct {
	Enabled             bool   `json:"enabled"`
	WorkspaceResourceID string `json:"workspaceResourceId,omitempty"`
	WorkspaceID         string `json:"workspaceId,omitempty"`
	WorkspaceKey        string `json:"workspaceKey,omitempty"`
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

	// Memory optimized VMs
	StandardE4sV3  VMSize = "Standard_E4s_v3"
	StandardE8sV3  VMSize = "Standard_E8s_v3"
	StandardE16sV3 VMSize = "Standard_E16s_v3"
	StandardE32sV3 VMSize = "Standard_E32s_v3"

	// Compute optimized VMs
	StandardF8sV2  VMSize = "Standard_F8s_v2"
	StandardF16sV2 VMSize = "Standard_F16s_v2"
	StandardF32sV2 VMSize = "Standard_F32s_v2"
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

// GenevaActionPluginVersion is the struct returned by the GetPluginVersion
// Geneva action API
type GenevaActionPluginVersion struct {
	PluginVersion *string `json:"pluginVersion,omitempty"`
}

// GenevaActionListClusterVMs is the struct returned by the ListClusterVMs
// Geneva action API
type GenevaActionListClusterVMs struct {
	VMs *[]string `json:"vms,omitempty"`
}

// GenevaActionListEtcdBackups is returned by the ListEtcdBackups Geneva action
// API
type GenevaActionListEtcdBackups struct {
	Name         string    `json:"name,omitempty"`
	LastModified time.Time `json:"lastModified,omitempty"`
}
