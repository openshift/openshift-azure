package api

import (
	"strings"
)

// OpenShiftCluster complies with the ARM model of
// resource definition in a JSON template.
type OpenShiftCluster struct {
	ID       string            `json:"id"`
	Location string            `json:"location"`
	Name     string            `json:"name"`
	Tags     map[string]string `json:"tags"`
	Type     string            `json:"type"`

	Properties *Properties `json:"properties,omitempty"`
}

type Properties struct {
	ProvisioningState   ProvisioningState    `json:"provisioningState,omitempty"`
	OrchestratorProfile *OrchestratorProfile `json:"orchestratorProfile,omitempty"`
	MasterProfile       *MasterProfile       `json:"masterProfile,omitempty"`
	AgentPoolProfiles   []*AgentPoolProfile  `json:"agentPoolProfiles,omitempty"`
	LinuxProfile        *LinuxProfile        `json:"linuxProfile,omitempty"`
	CertificateProfile  *CertificateProfile  `json:"certificateProfile,omitempty"`
	AADProfile          *AADProfile          `json:"aadProfile,omitempty"`
}

// CertificateProfile represents the definition of the master cluster
type CertificateProfile struct {
	// CaCertificate is the certificate authority certificate.
	CaCertificate string `json:"caCertificate,omitempty" conform:"redact"`
	// CaPrivateKey is the certificate authority key.
	CaPrivateKey string `json:"caPrivateKey,omitempty" conform:"redact"`
	// ApiServerCertificate is the rest api server certificate, and signed by the CA
	APIServerCertificate string `json:"apiServerCertificate,omitempty" conform:"redact"`
	// ApiServerPrivateKey is the rest api server private key, and signed by the CA
	APIServerPrivateKey string `json:"apiServerPrivateKey,omitempty" conform:"redact"`
	// ClientCertificate is the certificate used by the client kubelet services and signed by the CA
	ClientCertificate string `json:"clientCertificate,omitempty" conform:"redact"`
	// ClientPrivateKey is the private key used by the client kubelet services and signed by the CA
	ClientPrivateKey string `json:"clientPrivateKey,omitempty" conform:"redact"`
	// KubeConfigCertificate is the client certificate used for kubectl cli and signed by the CA
	KubeConfigCertificate string `json:"kubeConfigCertificate,omitempty" conform:"redact"`
	// KubeConfigPrivateKey is the client private key used for kubectl cli and signed by the CA
	KubeConfigPrivateKey string `json:"kubeConfigPrivateKey,omitempty" conform:"redact"`
	// EtcdServerCertificate is the server certificate for etcd, and signed by the CA
	EtcdServerCertificate string `json:"etcdServerCertificate,omitempty" conform:"redact"`
	// EtcdServerPrivateKey is the server private key for etcd, and signed by the CA
	EtcdServerPrivateKey string `json:"etcdServerPrivateKey,omitempty" conform:"redact"`
	// EtcdClientCertificate is etcd client certificate, and signed by the CA
	EtcdClientCertificate string `json:"etcdClientCertificate,omitempty" conform:"redact"`
	// EtcdClientPrivateKey is the etcd client private key, and signed by the CA
	EtcdClientPrivateKey string `json:"etcdClientPrivateKey,omitempty" conform:"redact"`
	// EtcdPeerCertificates is list of etcd peer certificates, and signed by the CA
	EtcdPeerCertificates []string `json:"etcdPeerCertificates,omitempty" conform:"redact"`
	// EtcdPeerPrivateKeys is list of etcd peer private keys, and signed by the CA
	EtcdPeerPrivateKeys []string `json:"etcdPeerPrivateKeys,omitempty" conform:"redact"`
}

// LinuxProfile represents the linux parameters passed to the cluster
type LinuxProfile struct {
	AdminUsername string `json:"adminUsername"`
	SSH           struct {
		PublicKeys []PublicKey `json:"publicKeys"`
	} `json:"ssh"`
	Secrets []KeyVaultSecrets `json:"secrets,omitempty"`
	Distro  Distro            `json:"distro,omitempty"`
}

// PublicKey represents an SSH key for LinuxProfile
type PublicKey struct {
	KeyData string `json:"keyData"`
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
	OpenShiftVersion string           `json:"openShiftVersion"`
	OpenShiftConfig  *OpenShiftConfig `json:"openshiftConfig,omitempty"`
}

// KubernetesContainerSpec defines configuration for a container spec
type KubernetesContainerSpec struct {
	Name           string `json:"name,omitempty"`
	Image          string `json:"image,omitempty"`
	CPURequests    string `json:"cpuRequests,omitempty"`
	MemoryRequests string `json:"memoryRequests,omitempty"`
	CPULimits      string `json:"cpuLimits,omitempty"`
	MemoryLimits   string `json:"memoryLimits,omitempty"`
}

// KubernetesAddon defines a list of addons w/ configuration to include with the cluster deployment
type KubernetesAddon struct {
	Name       string                    `json:"name,omitempty"`
	Enabled    *bool                     `json:"enabled,omitempty"`
	Containers []KubernetesContainerSpec `json:"containers,omitempty"`
	Config     map[string]string         `json:"config,omitempty"`
}

// IsEnabled returns if the addon is explicitly enabled, or the user-provided default if non explicitly enabled
func (a *KubernetesAddon) IsEnabled(ifNil bool) bool {
	if a.Enabled == nil {
		return ifNil
	}
	return *a.Enabled
}

// CloudProviderConfig contains the KubernetesConfig properties specific to the Cloud Provider
// TODO use this when strict JSON checking accommodates struct embedding
type CloudProviderConfig struct {
	CloudProviderBackoff         bool    `json:"cloudProviderBackoff,omitempty"`
	CloudProviderBackoffRetries  int     `json:"cloudProviderBackoffRetries,omitempty"`
	CloudProviderBackoffJitter   float64 `json:"cloudProviderBackoffJitter,omitempty"`
	CloudProviderBackoffDuration int     `json:"cloudProviderBackoffDuration,omitempty"`
	CloudProviderBackoffExponent float64 `json:"cloudProviderBackoffExponent,omitempty"`
	CloudProviderRateLimit       bool    `json:"cloudProviderRateLimit,omitempty"`
	CloudProviderRateLimitQPS    float64 `json:"cloudProviderRateLimitQPS,omitempty"`
	CloudProviderRateLimitBucket int     `json:"cloudProviderRateLimitBucket,omitempty"`
}

// KubernetesConfig contains the Kubernetes config structure, containing
// Kubernetes specific configuration
type KubernetesConfig struct {
	KubernetesImageBase              string            `json:"kubernetesImageBase,omitempty"`
	ClusterSubnet                    string            `json:"clusterSubnet,omitempty"`
	NetworkPolicy                    string            `json:"networkPolicy,omitempty"`
	NetworkPlugin                    string            `json:"networkPlugin,omitempty"`
	ContainerRuntime                 string            `json:"containerRuntime,omitempty"`
	MaxPods                          int               `json:"maxPods,omitempty"`
	DockerBridgeSubnet               string            `json:"dockerBridgeSubnet,omitempty"`
	DNSServiceIP                     string            `json:"dnsServiceIP,omitempty"`
	ServiceCIDR                      string            `json:"serviceCidr,omitempty"`
	UseManagedIdentity               bool              `json:"useManagedIdentity,omitempty"`
	CustomHyperkubeImage             string            `json:"customHyperkubeImage,omitempty"`
	DockerEngineVersion              string            `json:"dockerEngineVersion,omitempty"`
	CustomCcmImage                   string            `json:"customCcmImage,omitempty"` // Image for cloud-controller-manager
	UseCloudControllerManager        *bool             `json:"useCloudControllerManager,omitempty"`
	CustomWindowsPackageURL          string            `json:"customWindowsPackageURL,omitempty"`
	UseInstanceMetadata              *bool             `json:"useInstanceMetadata,omitempty"`
	EnableRbac                       *bool             `json:"enableRbac,omitempty"`
	EnableSecureKubelet              *bool             `json:"enableSecureKubelet,omitempty"`
	EnableAggregatedAPIs             bool              `json:"enableAggregatedAPIs,omitempty"`
	GCHighThreshold                  int               `json:"gchighthreshold,omitempty"`
	GCLowThreshold                   int               `json:"gclowthreshold,omitempty"`
	EtcdVersion                      string            `json:"etcdVersion,omitempty"`
	EtcdDiskSizeGB                   string            `json:"etcdDiskSizeGB,omitempty"`
	EtcdEncryptionKey                string            `json:"etcdEncryptionKey,omitempty"`
	EnableDataEncryptionAtRest       *bool             `json:"enableDataEncryptionAtRest,omitempty"`
	EnableEncryptionWithExternalKms  *bool             `json:"enableEncryptionWithExternalKms,omitempty"`
	EnablePodSecurityPolicy          *bool             `json:"enablePodSecurityPolicy,omitempty"`
	Addons                           []KubernetesAddon `json:"addons,omitempty"`
	KubeletConfig                    map[string]string `json:"kubeletConfig,omitempty"`
	ControllerManagerConfig          map[string]string `json:"controllerManagerConfig,omitempty"`
	CloudControllerManagerConfig     map[string]string `json:"cloudControllerManagerConfig,omitempty"`
	APIServerConfig                  map[string]string `json:"apiServerConfig,omitempty"`
	SchedulerConfig                  map[string]string `json:"schedulerConfig,omitempty"`
	CloudProviderBackoff             bool              `json:"cloudProviderBackoff,omitempty"`
	CloudProviderBackoffRetries      int               `json:"cloudProviderBackoffRetries,omitempty"`
	CloudProviderBackoffJitter       float64           `json:"cloudProviderBackoffJitter,omitempty"`
	CloudProviderBackoffDuration     int               `json:"cloudProviderBackoffDuration,omitempty"`
	CloudProviderBackoffExponent     float64           `json:"cloudProviderBackoffExponent,omitempty"`
	CloudProviderRateLimit           bool              `json:"cloudProviderRateLimit,omitempty"`
	CloudProviderRateLimitQPS        float64           `json:"cloudProviderRateLimitQPS,omitempty"`
	CloudProviderRateLimitBucket     int               `json:"cloudProviderRateLimitBucket,omitempty"`
	NonMasqueradeCidr                string            `json:"nonMasqueradeCidr,omitempty"`
	NodeStatusUpdateFrequency        string            `json:"nodeStatusUpdateFrequency,omitempty"`
	HardEvictionThreshold            string            `json:"hardEvictionThreshold,omitempty"`
	CtrlMgrNodeMonitorGracePeriod    string            `json:"ctrlMgrNodeMonitorGracePeriod,omitempty"`
	CtrlMgrPodEvictionTimeout        string            `json:"ctrlMgrPodEvictionTimeout,omitempty"`
	CtrlMgrRouteReconciliationPeriod string            `json:"ctrlMgrRouteReconciliationPeriod,omitempty"`
}

// CustomFile has source as the full absolute source path to a file and dest
// is the full absolute desired destination path to put the file on a master node
type CustomFile struct {
	Source string `json:"source,omitempty"`
	Dest   string `json:"dest,omitempty"`
}

// OpenShiftConfig holds configuration for OpenShift
type OpenShiftConfig struct {
	KubernetesConfig *KubernetesConfig `json:"kubernetesConfig,omitempty"`
	ConfigBundles    map[string][]byte `json:"configBundles,omitempty"`
}

// MasterProfile represents the definition of the master cluster
type MasterProfile struct {
	Count                    int             `json:"count"`
	DNSPrefix                string          `json:"dnsPrefix"`
	SubjectAltNames          []string        `json:"subjectAltNames"`
	VMSize                   string          `json:"vmSize"`
	OSDiskSizeGB             int             `json:"osDiskSizeGB,omitempty"`
	VnetSubnetID             string          `json:"vnetSubnetID,omitempty"`
	VnetCidr                 string          `json:"vnetCidr,omitempty"`
	FirstConsecutiveStaticIP string          `json:"firstConsecutiveStaticIP,omitempty"`
	Subnet                   string          `json:"subnet"`
	IPAddressCount           int             `json:"ipAddressCount,omitempty"`
	HTTPSourceAddressPrefix  string          `json:"HTTPSourceAddressPrefix,omitempty"`
	OAuthEnabled             bool            `json:"oauthEnabled"`
	Distro                   Distro          `json:"distro,omitempty"`
	ImageRef                 *ImageReference `json:"imageReference,omitempty"`
	CustomFiles              *[]CustomFile   `json:"customFiles,omitempty"`

	// Master LB public endpoint/FQDN with port
	// The format will be FQDN:2376
	// Not used during PUT, returned as part of GET
	FQDN string `json:"fqdn,omitempty"`
}

// ImageReference represents a reference to an Image resource in Azure.
type ImageReference struct {
	Name          string `json:"name,omitempty"`
	ResourceGroup string `json:"resourceGroup,omitempty"`
}

// AgentPoolProfile represents an agent pool definition
type AgentPoolProfile struct {
	Name                         string               `json:"name"`
	Count                        int                  `json:"count"`
	VMSize                       string               `json:"vmSize"`
	OSDiskSizeGB                 int                  `json:"osDiskSizeGB,omitempty"`
	OSType                       OSType               `json:"osType,omitempty"`
	Ports                        []int                `json:"ports,omitempty"`
	ScaleSetPriority             string               `json:"scaleSetPriority,omitempty"`
	ScaleSetEvictionPolicy       string               `json:"scaleSetEvictionPolicy,omitempty"`
	DiskSizesGB                  []int                `json:"diskSizesGB,omitempty"`
	VnetSubnetID                 string               `json:"vnetSubnetID,omitempty"`
	Subnet                       string               `json:"subnet"`
	IPAddressCount               int                  `json:"ipAddressCount,omitempty"`
	Distro                       Distro               `json:"distro,omitempty"`
	Role                         AgentPoolProfileRole `json:"role,omitempty"`
	AcceleratedNetworkingEnabled bool                 `json:"acceleratedNetworkingEnabled,omitempty"`
	CustomNodeLabels             map[string]string    `json:"customNodeLabels,omitempty"`
	KubernetesConfig             *KubernetesConfig    `json:"kubernetesConfig,omitempty"`
	ImageRef                     *ImageReference      `json:"imageReference,omitempty"`
}

// AgentPoolProfileRole represents an agent role
type AgentPoolProfileRole string

// KeyVaultSecrets specifies certificates to install on the pool
// of machines from a given key vault
// the key vault specified must have been granted read permissions to CRP
type KeyVaultSecrets struct {
	SourceVault       *KeyVaultID           `json:"sourceVault,omitempty"`
	VaultCertificates []KeyVaultCertificate `json:"vaultCertificates,omitempty"`
}

// KeyVaultID specifies a key vault
type KeyVaultID struct {
	ID string `json:"id,omitempty"`
}

// KeyVaultCertificate specifies a certificate to install
// On Linux, the certificate file is placed under the /var/lib/waagent directory
// with the file name <UppercaseThumbprint>.crt for the X509 certificate file
// and <UppercaseThumbprint>.prv for the private key. Both of these files are .pem formatted.
// On windows the certificate will be saved in the specified store.
type KeyVaultCertificate struct {
	CertificateURL   string `json:"certificateUrl,omitempty"`
	CertificateStore string `json:"certificateStore,omitempty"`
}

// OSType represents OS types of agents
type OSType string

// Distro represents Linux distro to use for Linux VMs
type Distro string

// AuthenticatorType represents the authenticator type the cluster was
// set up with.
type AuthenticatorType string

// AADProfile specifies attributes for AAD integration
type AADProfile struct {
	// The client AAD application ID.
	ClientAppID string `json:"clientAppID,omitempty"`
	// The server AAD application ID.
	ServerAppID string `json:"serverAppID,omitempty"`
	// The server AAD application secret
	ServerAppSecret string `json:"serverAppSecret,omitempty" conform:"redact"`
	// The AAD tenant ID to use for authentication.
	// If not specified, will use the tenant of the deployment subscription.
	// Optional
	TenantID string `json:"tenantID,omitempty"`
	// The authenticator to use, either "oidc" or "webhook".
	Authenticator AuthenticatorType `json:"authenticator"`
}

// TotalNodes returns the total number of nodes in the cluster configuration
func (p *Properties) TotalNodes() int {
	var totalNodes int
	if p.MasterProfile != nil {
		totalNodes = p.MasterProfile.Count
	}
	for _, pool := range p.AgentPoolProfiles {
		totalNodes = totalNodes + pool.Count
	}
	return totalNodes
}

// HasVirtualMachineScaleSets returns true if the cluster contains Virtual Machine Scale Sets
func (p *Properties) HasVirtualMachineScaleSets() bool {
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.AvailabilityProfile == VirtualMachineScaleSets {
			return true
		}
	}
	return false
}

// IsCustomVNET returns true if the customer brought their own VNET
func (m *MasterProfile) IsCustomVNET() bool {
	return len(m.VnetSubnetID) > 0
}

// IsRHEL returns true if the master specified a RHEL distro
func (m *MasterProfile) IsRHEL() bool {
	return m.Distro == RHEL
}

// IsCoreOS returns true if the master specified a CoreOS distro
func (m *MasterProfile) IsCoreOS() bool {
	return m.Distro == CoreOS
}

// IsCustomVNET returns true if the customer brought their own VNET
func (a *AgentPoolProfile) IsCustomVNET() bool {
	return len(a.VnetSubnetID) > 0
}

// IsLinux returns true if the agent pool is linux
func (a *AgentPoolProfile) IsLinux() bool {
	return a.OSType == Linux
}

// IsRHEL returns true if the agent pool specified a RHEL distro
func (a *AgentPoolProfile) IsRHEL() bool {
	return a.OSType == Linux && a.Distro == RHEL
}

// IsCoreOS returns true if the agent specified a CoreOS distro
func (a *AgentPoolProfile) IsCoreOS() bool {
	return a.OSType == Linux && a.Distro == CoreOS
}

// IsAvailabilitySets returns true if the customer specified disks
func (a *AgentPoolProfile) IsAvailabilitySets() bool {
	return a.AvailabilityProfile == AvailabilitySet
}

// IsVirtualMachineScaleSets returns true if the agent pool availability profile is VMSS
func (a *AgentPoolProfile) IsVirtualMachineScaleSets() bool {
	return a.AvailabilityProfile == VirtualMachineScaleSets
}

// IsLowPriorityScaleSet returns true if the VMSS is Low Priority
func (a *AgentPoolProfile) IsLowPriorityScaleSet() bool {
	return a.AvailabilityProfile == VirtualMachineScaleSets && a.ScaleSetPriority == ScaleSetPriorityLow
}

// HasDisks returns true if the customer specified disks
func (a *AgentPoolProfile) HasDisks() bool {
	return len(a.DiskSizesGB) > 0
}

// HasSecrets returns true if the customer specified secrets to install
func (l *LinuxProfile) HasSecrets() bool {
	return len(l.Secrets) > 0
}

// IsAzureCNI returns true if Azure CNI network plugin is enabled
func (o *OrchestratorProfile) IsAzureCNI() bool {
	if o.OpenShiftConfig != nil {
		return o.OpenShiftConfig.KubernetesConfig.NetworkPlugin == "azure"
	}
	return false
}

// RequireRouteTable returns true if this deployment requires routing table
func (o *OrchestratorProfile) RequireRouteTable() bool {
	if o.IsAzureCNI() || "cilium" == o.OpenShiftConfig.KubernetesConfig.NetworkPolicy {
		return false
	}
	return true
}

// HasAadProfile  returns true if the has aad profile
func (p *Properties) HasAadProfile() bool {
	return p.AADProfile != nil
}

func isNSeriesSKU(p *Properties) bool {
	for _, profile := range p.AgentPoolProfiles {
		if strings.Contains(profile.VMSize, "Standard_N") {
			return true
		}
	}
	return false
}

// PrivateJumpboxProvision checks if a private cluster has jumpbox auto-provisioning
func (k *KubernetesConfig) PrivateJumpboxProvision() bool {
	if k != nil && k.PrivateCluster != nil && *k.PrivateCluster.Enabled && k.PrivateCluster.JumpboxProfile != nil {
		return true
	}
	return false
}
