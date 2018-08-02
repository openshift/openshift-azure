package api

// the orchestrators supported by vlabs
const (
	// OpenShift is the string constant for the OpenShift orchestrator type
	OpenShift string = "OpenShift"
)

// the OSTypes supported by vlabs
const (
	Windows OSType = "Windows"
	Linux   OSType = "Linux"
)

// the LinuxDistros supported by vlabs
const (
	RHEL   Distro = "rhel"
	CoreOS Distro = "coreos"
	// Supported distros by OpenShift
	OpenShift39RHEL Distro = "openshift39_rhel"
	OpenShiftCentOS Distro = "openshift39_centos"
)

// Availability profiles
const (
	// ScaleSetPriorityRegular is the default ScaleSet Priority
	ScaleSetPriorityRegular = "Regular"
	// ScaleSetPriorityLow means the ScaleSet will use Low-priority VMs
	ScaleSetPriorityLow = "Low"
	// ScaleSetEvictionPolicyDelete is the default Eviction Policy for Low-priority VM ScaleSets
	ScaleSetEvictionPolicyDelete = "Delete"
	// ScaleSetEvictionPolicyDeallocate means a Low-priority VM ScaleSet will deallocate, rather than delete, VMs.
	ScaleSetEvictionPolicyDeallocate = "Deallocate"
)

const (
	// AgentPoolProfileRoleCompute is the compute role
	AgentPoolProfileRoleCompute AgentPoolProfileRole = "compute"
	// AgentPoolProfileRoleInfra is the infra role
	AgentPoolProfileRoleInfra AgentPoolProfileRole = "infra"
	// AgentPoolProfileRoleMaster is the master role
	AgentPoolProfileRoleMaster AgentPoolProfileRole = "master"
)

const (
	// OIDC represent cluster setup in OIDC auth mode
	OIDC AuthenticatorType = "oidc"
	// Webhook represent cluster setup in wehhook auth mode
	Webhook AuthenticatorType = "webhook"
)
