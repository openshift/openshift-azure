package constants

const (
	// main pluging ARM names/constants
	VnetName                                      = "vnet"
	VnetSubnetName                                = "default"
	VnetManagementSubnetName                      = "management"
	IPAPIServerName                               = "ip-apiserver"
	IPOutboundName                                = "ip-outbound"
	LbAPIServerName                               = "lb-apiserver"
	IlbAPIServerName                              = "lb-apiserver-internal"
	LbAPIServerFrontendConfigurationName          = "frontend"
	IlbAPIServerFrontendConfigurationName         = "lb-frontend-internal"
	LbAPIServerBackendPoolName                    = "backend"
	LbSSHLoadBalancingRuleName                    = "port-22"
	LbAPIServerLoadBalancingRuleName              = "port-443"
	LbAPIServerProbeName                          = "port-443"
	LbKubernetesName                              = "kubernetes" // must match KubeCloudSharedConfiguration ClusterName
	LbKubernetesOutboundFrontendConfigurationName = "outbound"
	LbKubernetesOutboundRuleName                  = "outbound"
	LbKubernetesBackendPoolName                   = "kubernetes" // must match KubeCloudSharedConfiguration ClusterName
	NsgMasterName                                 = "nsg-master"
	NsgMasterAllowSSHRuleName                     = "allow_ssh"
	NsgMasterAllowHTTPSRuleName                   = "allow_https"
	NsgWorkerName                                 = "nsg-worker"
	VmssNicName                                   = "nic"
	VmssNicPublicIPConfigurationName              = "ip"
	VmssIPConfigurationName                       = "ipconfig"
	VmssCSEName                                   = "cse"
	VmssAdminUsername                             = "cloud-user"
	VmssType                                      = "vmss"
	LoadBalancerSku                               = "standard"
)
