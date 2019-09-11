package vmimage

import (
	"bytes"
	"compress/gzip"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

var (
	// The versions referenced here must be kept in lockstep with the imports
	// above.
	versionMap = map[string]string{
		"Microsoft.Compute": "2018-10-01",
		"Microsoft.Network": "2018-07-01",
	}
)

const (
	vnetName       = "vnet"
	vnetSubnetName = "default"
	ipName         = "ip"
	nsgName        = "nsg"
	nicName        = "nic"
	vmName         = "vm"
	cseName        = "vm/cse"
	adminUsername  = "cloud-user"
)

func vnet(location string) *arm.VirtualNetwork {
	return &arm.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{
					"10.0.0.0/24",
				},
			},
			Subnets: &[]network.Subnet{
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr("10.0.0.0/24"),
					},
					Name: to.StringPtr(vnetSubnetName),
				},
			},
		},
		Name:     to.StringPtr(vnetName),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
		Location: to.StringPtr(location),
	}
}

func ip(resourcegroup, location, domainNameLabel string) *arm.PublicIPAddress {
	return &arm.PublicIPAddress{
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuNameBasic,
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Dynamic,
			DNSSettings: &network.PublicIPAddressDNSSettings{
				DomainNameLabel: to.StringPtr(domainNameLabel),
			},
		},
		Name:     to.StringPtr(ipName),
		Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
		Location: to.StringPtr(location),
	}
}

func nsg(location string) *arm.SecurityGroup {
	return &arm.SecurityGroup{
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
			},
		},
		Name:     to.StringPtr(nsgName),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		Location: to.StringPtr(location),
	}
}

func nic(subscriptionID, resourceGroup, location string) *arm.Interface {
	return &arm.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: to.StringPtr(resourceid.ResourceID(
					subscriptionID,
					resourceGroup,
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
								subscriptionID,
								resourceGroup,
								"Microsoft.Network/virtualNetworks",
								vnetName,
							) + "/subnets/" + vnetSubnetName),
						},
						PublicIPAddress: &network.PublicIPAddress{
							ID: to.StringPtr(resourceid.ResourceID(
								subscriptionID,
								resourceGroup,
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
		Location: to.StringPtr(location),
	}
}

func vm(subscriptionID, resourceGroup, location, sshPublicKey string, plan *compute.Plan, imageReference *compute.ImageReference) *arm.VirtualMachine {
	return &arm.VirtualMachine{
		Plan: plan,
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
							subscriptionID,
							resourceGroup,
							"Microsoft.Network/networkInterfaces",
							nicName,
						)),
					},
				},
			},
		},
		Name:     to.StringPtr(vmName),
		Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
		Location: to.StringPtr(location),
	}
}

func cse(location string, script []byte) (*arm.VirtualMachineExtension, error) {
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

	return &arm.VirtualMachineExtension{
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
		Location: to.StringPtr(location),
	}, nil
}
