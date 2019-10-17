package proxyinfrastructure

import (
	"bytes"
	"compress/gzip"
	"os"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

func vnet(conf *Config) *network.VirtualNetwork {
	return &network.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{
					conf.NetDefinition.Vnet,
				},
			},
			Subnets: &[]network.Subnet{
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(conf.NetDefinition.DefaultSubnet),
					},
					Name: to.StringPtr(vnetSubnetName),
				},
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(conf.NetDefinition.ManagementSubnet),
					},
					Name: to.StringPtr(vnetManagementSubnetName),
				},
			},
		},
		Name:     to.StringPtr(vnetName),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
		Location: to.StringPtr(conf.region),
	}
}

func ip(conf *Config) *network.PublicIPAddress {
	return &network.PublicIPAddress{
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuNameBasic,
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Dynamic,
		},
		Name:     to.StringPtr(ipName),
		Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
		Location: to.StringPtr(conf.region),
	}
}

func nsg(conf *Config) *network.SecurityGroup {
	return &network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &[]network.SecurityRule{
				{
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourcePortRange:          to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("22"),
						SourceAddressPrefix:      to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(100),
						Direction:                network.SecurityRuleDirectionInbound,
					},
					Name: to.StringPtr("ssh"),
				},
				{
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourcePortRange:          to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("8443"),
						SourceAddressPrefix:      to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(101),
						Direction:                network.SecurityRuleDirectionInbound,
					},
					Name: to.StringPtr("proxy"),
				},
			},
		},
		Name:     to.StringPtr(nsgName),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		Location: to.StringPtr(conf.region),
	}
}

func nic(conf *Config) *network.Interface {
	return &network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: to.StringPtr(resourceid.ResourceID(
					os.Getenv("AZURE_SUBSCRIPTION_ID"),
					conf.resourceGroup,
					"Microsoft.Network/networkSecurityGroups",
					nsgName,
				)),
			},
			IPConfigurations: &[]network.InterfaceIPConfiguration{
				{
					InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: network.Dynamic,
						Subnet: &network.Subnet{
							ID: to.StringPtr(resourceid.ResourceID(
								os.Getenv("AZURE_SUBSCRIPTION_ID"),
								conf.resourceGroup,
								"Microsoft.Network/virtualNetworks",
								vnetName,
							) + "/subnets/" + vnetSubnetName),
						},
						PublicIPAddress: &network.PublicIPAddress{
							ID: to.StringPtr(resourceid.ResourceID(
								os.Getenv("AZURE_SUBSCRIPTION_ID"),
								conf.resourceGroup,
								"Microsoft.Network/publicIpAddresses",
								ipName,
							)),
						},
					},
					Name: to.StringPtr("ipconfig"),
				},
			},
		},
		Name:     to.StringPtr(nicName),
		Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
		Location: to.StringPtr(conf.region),
	}
}

func vmImageReference() *compute.ImageReference {
	return &compute.ImageReference{
		Publisher: to.StringPtr("RedHat"),
		Offer:     to.StringPtr("RHEL"),
		Sku:       to.StringPtr("7-RAW"),
		Version:   to.StringPtr("latest"),
	}
}

func vm(conf *Config) *compute.VirtualMachine {
	imageReference := vmImageReference()
	sshPublicKey, err := tls.SSHPublicKeyAsString(&conf.sshKey.PublicKey)
	if err != nil {
		panic(err)
	}
	return &compute.VirtualMachine{
		//Plan: &compute.Plan{
		//	Name:      imageReference.Sku,
		//	Publisher: imageReference.Publisher,
		//	Product:   imageReference.Offer,
		//},
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypesStandardD4sV3,
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: imageReference,
				OsDisk: &compute.OSDisk{
					CreateOption: compute.DiskCreateOptionTypesFromImage,
					ManagedDisk: &compute.ManagedDiskParameters{
						StorageAccountType: compute.StorageAccountTypesPremiumLRS,
					},
					DiskSizeGB: to.Int32Ptr(64),
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  to.StringPtr("vm"),
				AdminUsername: to.StringPtr(adminUsername),
				LinuxConfiguration: &compute.LinuxConfiguration{
					DisablePasswordAuthentication: to.BoolPtr(true),
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								Path:    to.StringPtr("/home/" + adminUsername + "/.ssh/authorized_keys"),
								KeyData: to.StringPtr(sshPublicKey),
							},
						},
					},
				},
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: to.StringPtr(resourceid.ResourceID(
							os.Getenv("AZURE_SUBSCRIPTION_ID"),
							conf.resourceGroup,
							"Microsoft.Network/networkInterfaces",
							nicName,
						)),
					},
				},
			},
		},
		Name:     to.StringPtr(vmName),
		Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
		Location: to.StringPtr(conf.region),
	}
}

func cse(conf *Config, script []byte) (*compute.VirtualMachineExtension, error) {
	buf := &bytes.Buffer{}

	gz, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	_, err = gz.Write(script)
	if err != nil {
		return nil, err
	}

	err = gz.Close()
	if err != nil {
		return nil, err
	}

	return &compute.VirtualMachineExtension{
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
			Type:                    to.StringPtr("CustomScript"),
			TypeHandlerVersion:      to.StringPtr("2.0"),
			AutoUpgradeMinorVersion: to.BoolPtr(true),
			ProtectedSettings: map[string]interface{}{
				"script": buf.Bytes(),
			},
		},
		Name:     to.StringPtr(cseName),
		Type:     to.StringPtr("Microsoft.Compute/virtualMachines/extensions"),
		Location: to.StringPtr(conf.region),
	}, nil
}
