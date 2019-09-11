package arm

import (
	"encoding/base64"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/template"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

var (
	// The versions referenced here must be kept in lockstep with the imports
	// above.
	versionMap = map[string]string{
		"Microsoft.Compute": "2018-10-01",
		"Microsoft.Network": "2018-07-01",
		"Microsoft.Storage": "2018-02-01",
	}
)

const (
	vnetName                                      = "vnet"
	vnetSubnetName                                = "default"
	ipAPIServerName                               = "ip-apiserver"
	ipOutboundName                                = "ip-outbound"
	lbAPIServerName                               = "lb-apiserver"
	lbAPIServerFrontendConfigurationName          = "frontend"
	lbAPIServerBackendPoolName                    = "backend"
	lbAPIServerLoadBalancingRuleName              = "port-443"
	lbAPIServerProbeName                          = "port-443"
	lbKubernetesName                              = "kubernetes" // must match KubeCloudSharedConfiguration ClusterName
	lbKubernetesOutboundFrontendConfigurationName = "outbound"
	lbKubernetesOutboundRuleName                  = "outbound"
	lbKubernetesBackendPoolName                   = "kubernetes" // must match KubeCloudSharedConfiguration ClusterName
	nsgMasterName                                 = "nsg-master"
	nsgMasterAllowSSHRuleName                     = "allow_ssh"
	nsgMasterAllowHTTPSRuleName                   = "allow_https"
	nsgWorkerName                                 = "nsg-worker"
	vmssNicName                                   = "nic"
	vmssNicPublicIPConfigurationName              = "ip"
	vmssIPConfigurationName                       = "ipconfig"
	vmssCSEName                                   = "cse"
	vmssAdminUsername                             = "cloud-user"
)

func (g *simpleGenerator) vnet() *arm.VirtualNetwork {
	return &arm.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{
					g.cs.Properties.NetworkProfile.VnetCIDR,
				},
			},
			Subnets: &[]network.Subnet{
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(g.cs.Properties.AgentPoolProfiles[0].SubnetCIDR),
					},
					Name: to.StringPtr(vnetSubnetName),
				},
			},
		},
		Name:     to.StringPtr(vnetName),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
		Location: to.StringPtr(g.cs.Location),
	}
}

func (g *simpleGenerator) ipAPIServer() *arm.PublicIPAddress {
	return &arm.PublicIPAddress{
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuNameStandard,
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Static,
			DNSSettings: &network.PublicIPAddressDNSSettings{
				DomainNameLabel: to.StringPtr(derived.MasterLBCNamePrefix(g.cs)),
			},
			IdleTimeoutInMinutes: to.Int32Ptr(15),
		},
		Name:     to.StringPtr(ipAPIServerName),
		Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
		Location: to.StringPtr(g.cs.Location),
	}
}

func (g *simpleGenerator) ipOutbound() *arm.PublicIPAddress {
	return &arm.PublicIPAddress{
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuNameStandard,
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Static,
			IdleTimeoutInMinutes:     to.Int32Ptr(15),
		},
		Name:     to.StringPtr(ipOutboundName),
		Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
		Location: to.StringPtr(g.cs.Location),
	}
}

func (g *simpleGenerator) lbAPIServer() *arm.LoadBalancer {
	lb := &arm.LoadBalancer{
		Sku: &network.LoadBalancerSku{
			Name: network.LoadBalancerSkuNameStandard,
		},
		LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
				{
					FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress: &network.PublicIPAddress{
							ID: to.StringPtr(resourceid.ResourceID(
								g.cs.Properties.AzProfile.SubscriptionID,
								g.cs.Properties.AzProfile.ResourceGroup,
								"Microsoft.Network/publicIPAddresses",
								ipAPIServerName,
							)),
						},
					},
					Name: to.StringPtr(lbAPIServerFrontendConfigurationName),
				},
			},
			BackendAddressPools: &[]network.BackendAddressPool{
				{
					Name: to.StringPtr(lbAPIServerBackendPoolName),
				},
			},
			LoadBalancingRules: &[]network.LoadBalancingRule{
				{
					LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
						FrontendIPConfiguration: &network.SubResource{
							ID: to.StringPtr(resourceid.ResourceID(
								g.cs.Properties.AzProfile.SubscriptionID,
								g.cs.Properties.AzProfile.ResourceGroup,
								"Microsoft.Network/loadBalancers",
								lbAPIServerName,
							) + "/frontendIPConfigurations/" + lbAPIServerFrontendConfigurationName),
						},
						BackendAddressPool: &network.SubResource{
							ID: to.StringPtr(resourceid.ResourceID(
								g.cs.Properties.AzProfile.SubscriptionID,
								g.cs.Properties.AzProfile.ResourceGroup,
								"Microsoft.Network/loadBalancers",
								lbAPIServerName,
							) + "/backendAddressPools/" + lbAPIServerBackendPoolName),
						},
						Probe: &network.SubResource{
							ID: to.StringPtr(resourceid.ResourceID(
								g.cs.Properties.AzProfile.SubscriptionID,
								g.cs.Properties.AzProfile.ResourceGroup,
								"Microsoft.Network/loadBalancers",
								lbAPIServerName,
							) + "/probes/" + lbAPIServerProbeName),
						},
						Protocol:             network.TransportProtocolTCP,
						LoadDistribution:     network.Default,
						FrontendPort:         to.Int32Ptr(443),
						BackendPort:          to.Int32Ptr(443),
						IdleTimeoutInMinutes: to.Int32Ptr(15),
						EnableFloatingIP:     to.BoolPtr(false),
					},
					Name: to.StringPtr(lbAPIServerLoadBalancingRuleName),
				},
			},
			Probes: &[]network.Probe{
				{
					ProbePropertiesFormat: &network.ProbePropertiesFormat{
						Protocol:          network.ProbeProtocolHTTPS,
						Port:              to.Int32Ptr(443),
						IntervalInSeconds: to.Int32Ptr(5),
						NumberOfProbes:    to.Int32Ptr(2),
						RequestPath:       to.StringPtr("/healthz"),
					},
					Name: to.StringPtr(lbAPIServerProbeName),
				},
			},
			InboundNatRules: &[]network.InboundNatRule{},
			InboundNatPools: &[]network.InboundNatPool{},
			OutboundRules:   &[]network.OutboundRule{},
		},
		Name:     to.StringPtr(lbAPIServerName),
		Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
		Location: to.StringPtr(g.cs.Location),
	}

	return lb
}

func (g *simpleGenerator) lbKubernetes() *arm.LoadBalancer {
	lb := &arm.LoadBalancer{
		Sku: &network.LoadBalancerSku{
			Name: network.LoadBalancerSkuNameStandard,
		},
		LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
				{
					FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress: &network.PublicIPAddress{
							ID: to.StringPtr(resourceid.ResourceID(
								g.cs.Properties.AzProfile.SubscriptionID,
								g.cs.Properties.AzProfile.ResourceGroup,
								"Microsoft.Network/publicIPAddresses",
								ipOutboundName,
							)),
						},
					},
					Name: to.StringPtr(lbKubernetesOutboundFrontendConfigurationName),
				},
			},
			BackendAddressPools: &[]network.BackendAddressPool{
				{
					Name: to.StringPtr(lbKubernetesBackendPoolName),
				},
			},
			OutboundRules: &[]network.OutboundRule{
				{
					Name: to.StringPtr(lbKubernetesOutboundRuleName),
					OutboundRulePropertiesFormat: &network.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: &[]network.SubResource{
							{
								ID: to.StringPtr(resourceid.ResourceID(
									g.cs.Properties.AzProfile.SubscriptionID,
									g.cs.Properties.AzProfile.ResourceGroup,
									"Microsoft.Network/loadBalancers",
									lbKubernetesName,
								) + "/frontendIPConfigurations/" + lbKubernetesOutboundFrontendConfigurationName),
							},
						},
						BackendAddressPool: &network.SubResource{
							ID: to.StringPtr(resourceid.ResourceID(
								g.cs.Properties.AzProfile.SubscriptionID,
								g.cs.Properties.AzProfile.ResourceGroup,
								"Microsoft.Network/loadBalancers",
								lbKubernetesName,
							) + "/backendAddressPools/" + lbKubernetesBackendPoolName),
						},
						Protocol:             network.Protocol1All,
						IdleTimeoutInMinutes: to.Int32Ptr(15),
					},
				},
			},
		},
		Name:     to.StringPtr(lbKubernetesName),
		Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
		Location: to.StringPtr(g.cs.Location),
	}

	return lb
}

func (g *simpleGenerator) storageAccount(name string, tags map[string]*string) *arm.Account {
	return &arm.Account{
		Sku: &storage.Sku{
			Name: storage.StandardLRS,
		},
		Kind:     storage.Storage,
		Name:     to.StringPtr(name),
		Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
		Location: to.StringPtr(g.cs.Location),
		Tags:     tags,
		AccountProperties: &storage.AccountProperties{
			EnableHTTPSTrafficOnly: to.BoolPtr(true),
		},
	}
}

func (g *simpleGenerator) nsgMaster() *arm.SecurityGroup {
	return &arm.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &[]network.SecurityRule{
				{
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Description:              to.StringPtr("Allow SSH traffic"),
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourcePortRange:          to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("22-22"),
						SourceAddressPrefixes:    to.StringSlicePtr(g.cs.Config.SSHSourceAddressPrefixes),
						DestinationAddressPrefix: to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(101),
						Direction:                network.SecurityRuleDirectionInbound,
					},
					Name: to.StringPtr(nsgMasterAllowSSHRuleName),
				},
				{
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Description:              to.StringPtr("Allow HTTPS traffic"),
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourcePortRange:          to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("443-443"),
						SourceAddressPrefixes:    to.StringSlicePtr([]string{"0.0.0.0/0"}),
						DestinationAddressPrefix: to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(102),
						Direction:                network.SecurityRuleDirectionInbound,
					},
					Name: to.StringPtr(nsgMasterAllowHTTPSRuleName),
				},
			},
		},
		Name:     to.StringPtr(nsgMasterName),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		Location: to.StringPtr(g.cs.Location),
	}
}

func (g *simpleGenerator) nsgWorker() *arm.SecurityGroup {
	return &arm.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &[]network.SecurityRule{},
		},
		Name:     to.StringPtr(nsgWorkerName),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		Location: to.StringPtr(g.cs.Location),
	}
}

func (g *simpleGenerator) Vmss(app *api.AgentPoolProfile, backupBlob, suffix string) (*compute.VirtualMachineScaleSet, error) {
	r, err := vmss(g.cs, app, backupBlob, suffix, g.testConfig)
	if err != nil {
		return nil, err
	}

	rr := compute.VirtualMachineScaleSet(*r)
	return &rr, err
}

func vmss(cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, backupBlob, suffix string, testConfig api.TestConfig) (*arm.VirtualMachineScaleSet, error) {
	sshPublicKey, err := tls.SSHPublicKeyAsString(&cs.Config.SSHKey.PublicKey)
	if err != nil {
		return nil, err
	}

	masterStartup, err := Asset("master-startup.sh")
	if err != nil {
		return nil, err
	}

	nodeStartup, err := Asset("node-startup.sh")
	if err != nil {
		return nil, err
	}

	var script string
	if app.Role == api.AgentPoolProfileRoleMaster {
		b, err := template.Template("master-startup.sh", string(masterStartup), nil, map[string]interface{}{
			"Config":         &cs.Config,
			"BackupBlobName": backupBlob,
		})
		if err != nil {
			return nil, err
		}
		script = base64.StdEncoding.EncodeToString(b)
	} else {
		b, err := template.Template("node-startup.sh", string(nodeStartup), nil, map[string]interface{}{
			"Config": &cs.Config,
			"Role":   app.Role,
		})
		if err != nil {
			return nil, err
		}
		script = base64.StdEncoding.EncodeToString(b)
	}

	vmss := &arm.VirtualMachineScaleSet{
		Sku: &compute.Sku{
			Name:     to.StringPtr(string(app.VMSize)),
			Tier:     to.StringPtr("Standard"),
			Capacity: to.Int64Ptr(app.Count),
		},
		Plan: &compute.Plan{
			Name:      to.StringPtr(cs.Config.ImageSKU),
			Publisher: to.StringPtr(cs.Config.ImagePublisher),
			Product:   to.StringPtr(cs.Config.ImageOffer),
		},
		VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
			UpgradePolicy: &compute.UpgradePolicy{
				AutomaticOSUpgradePolicy: &compute.AutomaticOSUpgradePolicy{
					DisableAutomaticRollback: to.BoolPtr(false),
				},
				Mode: compute.Manual,
			},
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
				OsProfile: &compute.VirtualMachineScaleSetOSProfile{
					ComputerNamePrefix: to.StringPtr(names.GetHostnamePrefix(app, suffix)),
					AdminUsername:      to.StringPtr(vmssAdminUsername),
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(true),
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									Path:    to.StringPtr("/home/" + vmssAdminUsername + "/.ssh/authorized_keys"),
									KeyData: to.StringPtr(sshPublicKey),
								},
							},
						},
					},
				},
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr(cs.Config.ImagePublisher),
						Offer:     to.StringPtr(cs.Config.ImageOffer),
						Sku:       to.StringPtr(cs.Config.ImageSKU),
						Version:   to.StringPtr(cs.Config.ImageVersion),
					},
					OsDisk: &compute.VirtualMachineScaleSetOSDisk{
						Caching:      compute.CachingTypesReadWrite,
						CreateOption: compute.DiskCreateOptionTypesFromImage,
						ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParameters{
							StorageAccountType: compute.StorageAccountTypesPremiumLRS,
						},
					},
				},
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
						{
							Name: to.StringPtr(vmssNicName),
							VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
								Primary: to.BoolPtr(true),
								IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
									{
										Name: to.StringPtr(vmssIPConfigurationName),
										VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
											Subnet: &compute.APIEntityReference{
												ID: to.StringPtr(resourceid.ResourceID(
													cs.Properties.AzProfile.SubscriptionID,
													cs.Properties.AzProfile.ResourceGroup,
													"Microsoft.Network/virtualNetworks",
													vnetName,
												) + "/subnets/" + vnetSubnetName),
											},
											Primary: to.BoolPtr(true),
										},
									},
								},
								EnableIPForwarding: to.BoolPtr(true),
							},
						},
					},
				},
				DiagnosticsProfile: &compute.DiagnosticsProfile{
					BootDiagnostics: &compute.BootDiagnostics{
						Enabled:    to.BoolPtr(true),
						StorageURI: to.StringPtr(fmt.Sprintf("https://%s.blob.core.windows.net", cs.Config.ConfigStorageAccount)),
					},
				},
				ExtensionProfile: &compute.VirtualMachineScaleSetExtensionProfile{
					Extensions: &[]compute.VirtualMachineScaleSetExtension{
						{
							Name: to.StringPtr(vmssCSEName),
							VirtualMachineScaleSetExtensionProperties: &compute.VirtualMachineScaleSetExtensionProperties{
								Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
								Type:                    to.StringPtr("CustomScript"),
								TypeHandlerVersion:      to.StringPtr("2.0"),
								AutoUpgradeMinorVersion: to.BoolPtr(true),
								Settings:                map[string]interface{}{},
								ProtectedSettings: map[string]interface{}{
									"script": script,
								},
							},
						},
					},
				},
			},
			SinglePlacementGroup: to.BoolPtr(false),
			Overprovision:        to.BoolPtr(false),
		},
		Name:     to.StringPtr(names.GetScalesetName(app, suffix)),
		Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
		Location: to.StringPtr(cs.Location),
	}

	if app.Role == api.AgentPoolProfileRoleMaster {
		vmss.VirtualMachineProfile.StorageProfile.DataDisks = &[]compute.VirtualMachineScaleSetDataDisk{
			{
				Lun:          to.Int32Ptr(0),
				Caching:      compute.CachingTypesReadOnly,
				CreateOption: compute.DiskCreateOptionTypesEmpty,
				DiskSizeGB:   to.Int32Ptr(256),
			},
		}
		(*(*vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations)[0].VirtualMachineScaleSetNetworkConfigurationProperties.IPConfigurations)[0].PublicIPAddressConfiguration = &compute.VirtualMachineScaleSetPublicIPAddressConfiguration{
			Name: to.StringPtr(vmssNicPublicIPConfigurationName),
			VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &compute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
				IdleTimeoutInMinutes: to.Int32Ptr(15),
			},
		}
		(*(*vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations)[0].VirtualMachineScaleSetNetworkConfigurationProperties.IPConfigurations)[0].LoadBalancerBackendAddressPools = &[]compute.SubResource{
			{
				ID: to.StringPtr(resourceid.ResourceID(
					cs.Properties.AzProfile.SubscriptionID,
					cs.Properties.AzProfile.ResourceGroup,
					"Microsoft.Network/loadBalancers",
					lbAPIServerName,
				) + "/backendAddressPools/" + lbAPIServerBackendPoolName),
			},
		}
		(*vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations)[0].VirtualMachineScaleSetNetworkConfigurationProperties.NetworkSecurityGroup = &compute.SubResource{
			ID: to.StringPtr(resourceid.ResourceID(
				cs.Properties.AzProfile.SubscriptionID,
				cs.Properties.AzProfile.ResourceGroup,
				"Microsoft.Network/networkSecurityGroups",
				nsgMasterName,
			)),
		}
	} else {
		(*(*vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations)[0].VirtualMachineScaleSetNetworkConfigurationProperties.IPConfigurations)[0].LoadBalancerBackendAddressPools = &[]compute.SubResource{
			{
				ID: to.StringPtr(resourceid.ResourceID(
					cs.Properties.AzProfile.SubscriptionID,
					cs.Properties.AzProfile.ResourceGroup,
					"Microsoft.Network/loadBalancers",
					lbKubernetesName,
				) + "/backendAddressPools/" + lbKubernetesBackendPoolName),
			},
		}
		(*vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations)[0].VirtualMachineScaleSetNetworkConfigurationProperties.NetworkSecurityGroup = &compute.SubResource{
			ID: to.StringPtr(resourceid.ResourceID(
				cs.Properties.AzProfile.SubscriptionID,
				cs.Properties.AzProfile.ResourceGroup,
				"Microsoft.Network/networkSecurityGroups",
				nsgWorkerName,
			)),
		}
	}

	if testConfig.ImageResourceName != "" {
		vmss.Plan = nil
		vmss.VirtualMachineScaleSetProperties.VirtualMachineProfile.StorageProfile.ImageReference = &compute.ImageReference{
			ID: to.StringPtr(resourceid.ResourceID(
				cs.Properties.AzProfile.SubscriptionID,
				testConfig.ImageResourceGroup,
				"Microsoft.Compute/images",
				testConfig.ImageResourceName,
			)),
		}
	}

	return vmss, nil
}
