package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	armconst "github.com/openshift/openshift-azure/pkg/arm/constants"
	fakerparm "github.com/openshift/openshift-azure/pkg/fakerp/arm"
	farmconst "github.com/openshift/openshift-azure/pkg/fakerp/arm/constants"
	"github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

type cidrName string

const (
	cidrVnet            cidrName = "cidrVnet"
	cidrDefaultSubnet   cidrName = "cidrDefaultSubnet"
	cidrGatewaySubnet   cidrName = "cidrGatewaySubnet"
	cidrManagmentSubnet cidrName = "cidrManagmentSubnet"
	cidrClients         cidrName = "cidrClients"
)

var (
	versionMap = map[string]string{
		"Microsoft.Network": "2019-04-01",
	}

	// subnets split logic:
	// vnet - contains all network addresses used for manamagement infrastructure.
	// at the moment it has 1024 addresses allocated.
	// x.x.0.0/22 - vnet network size
	// 	x.x.1.0/24 - subnet for the gateway
	//  x.x.2.0/24 - management subnet, where all EP/PLS resources should be created
	//  x.x.3.0/24 - reserved for future extensions
	// x.x.4.0/24 - out of the vnet subnet for VPN clients.
	subnets = []map[cidrName]string{
		{
			cidrVnet:            "172.30.0.0/22",
			cidrDefaultSubnet:   "172.30.0.0/24",
			cidrGatewaySubnet:   "172.30.1.0/24",
			cidrManagmentSubnet: "172.30.2.0/24",
			cidrClients:         "172.30.4.0/24",
		},
		{
			cidrVnet:            "172.30.8.0/22",
			cidrDefaultSubnet:   "172.30.8.0/24",
			cidrGatewaySubnet:   "172.30.9.0/24",
			cidrManagmentSubnet: "172.30.10.0/24",
			cidrClients:         "172.30.12.0/24",
		},
		{
			cidrVnet:            "172.30.16.0/22",
			cidrDefaultSubnet:   "172.30.16.0/24",
			cidrGatewaySubnet:   "172.30.17.0/24",
			cidrManagmentSubnet: "172.30.18.0/24",
			cidrClients:         "172.30.20.0/24",
		},
	}
)

type Config struct {
	SubscriptionID string   `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	TenantID       string   `envconfig:"AZURE_TENANT_ID" required:"true"`
	ClientID       string   `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret   string   `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	Regions        []string `envconfig:"AZURE_REGIONS" required:"true"`

	resourceGroup string
	subnets       map[cidrName]string
	region        string
}

func run(ctx context.Context, log *logrus.Entry) error {
	conf, err := newConfig(log)
	if err != nil {
		log.Fatal(err)
	}

	for i, region := range conf.Regions[:3] {
		conf.resourceGroup = fmt.Sprintf("%s-%s", "management", region)
		conf.subnets = subnets[i]
		conf.region = region

		// create resource groups for mangamenet vnets
		err = ensureResourceGroup(log, conf)
		if err != nil {
			return err
		}

		err = ensureResources(log, conf)
		if err != nil {
			return err
		}

	}

	return nil
}

// azureclient creates a resource group and returns whether the
// resource group was actually created or not and any error encountered.
func ensureResourceGroup(log *logrus.Entry, conf *Config) error {
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID, "")
	if err != nil {
		return err
	}
	ctx := context.Background()
	gc := resources.NewGroupsClient(ctx, log, conf.SubscriptionID, authorizer)

	if _, err := gc.Get(ctx, conf.resourceGroup); err == nil {
		return nil
	}

	_, err = gc.CreateOrUpdate(ctx, conf.resourceGroup, azresources.Group{Location: &conf.region})

	return err
}

// ensureResources creates a resources and returns whether the
// resources were actually created or not and any error encountered.
func ensureResources(log *logrus.Entry, conf *Config) error {
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID, "")
	if err != nil {
		return err
	}
	ctx := context.Background()
	deployments := resources.NewDeploymentsClient(ctx, log, conf.SubscriptionID, authorizer)

	template, err := generate(ctx, conf)
	if err != nil {
		return err
	}
	future, err := deployments.CreateOrUpdate(ctx, conf.resourceGroup, "azuredeploy", azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template: template,
			Mode:     azresources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	log.Info("waiting for arm template deployment to complete")
	err = future.WaitForCompletionRef(ctx, deployments.Client())
	if err != nil {
		log.Warnf("deployment failed: %#v", err)
	}

	return nil
}

// Generate generates fakeRP callback function objects for. This function mocks realRP
// impementation for required objects
func generate(ctx context.Context, conf *Config) (map[string]interface{}, error) {
	resources := []interface{}{
		vnet(conf),
		ipAddress(conf),
		virtualGateway(conf),
	}

	template, err := fakerparm.Generate(ctx, conf.SubscriptionID, conf.resourceGroup, resources)
	if err != nil {
		return nil, err
	}

	arm.FixupAPIVersions(template, versionMap)
	arm.FixupSDKMismatch(template)

	return template, nil
}

func vnet(conf *Config) *network.VirtualNetwork {
	return &network.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{
					conf.subnets[cidrVnet],
				},
			},
			Subnets: &[]network.Subnet{
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(conf.subnets[cidrDefaultSubnet]),
					},
					Name: to.StringPtr(armconst.VnetSubnetName),
				},
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(conf.subnets[cidrManagmentSubnet]),
					},
					Name: to.StringPtr(armconst.VnetManagementSubnetName),
				},
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr(conf.subnets[cidrGatewaySubnet]),
					},
					Name: to.StringPtr(farmconst.VnetGatewaySubnetName),
				},
			},
		},
		Name:     to.StringPtr(armconst.VnetName),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
		Location: to.StringPtr(conf.region),
	}
}

func ipAddress(conf *Config) *network.PublicIPAddress {
	return &network.PublicIPAddress{
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Dynamic,
			IdleTimeoutInMinutes:     to.Int32Ptr(15),
		},
		Name:     to.StringPtr(farmconst.GatewayIPAddressName),
		Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
		Location: to.StringPtr(conf.region),
	}
}

func virtualGateway(conf *Config) *network.VirtualNetworkGateway {
	cert, err := ioutil.ReadFile("secrets/vpn-rootCA.der")
	if err != nil {
		panic(err)
	}
	return &network.VirtualNetworkGateway{
		VirtualNetworkGatewayPropertiesFormat: &network.VirtualNetworkGatewayPropertiesFormat{
			GatewayType: network.VirtualNetworkGatewayTypeVpn,
			VpnType:     network.RouteBased,

			Sku: &network.VirtualNetworkGatewaySku{
				Tier: network.VirtualNetworkGatewaySkuTierVpnGw1,
				Name: network.VirtualNetworkGatewaySkuNameVpnGw1,
			},
			VpnClientConfiguration: &network.VpnClientConfiguration{
				VpnClientProtocols: &[]network.VpnClientProtocol{
					network.OpenVPN,
				},
				VpnClientAddressPool: &network.AddressSpace{
					AddressPrefixes: &[]string{conf.subnets[cidrClients]},
				},
				VpnClientRootCertificates: &[]network.VpnClientRootCertificate{
					{
						Name: to.StringPtr("management-root"),
						VpnClientRootCertificatePropertiesFormat: &network.VpnClientRootCertificatePropertiesFormat{
							PublicCertData: to.StringPtr(string(cert)),
						},
					},
				},
			},
			IPConfigurations: &[]network.VirtualNetworkGatewayIPConfiguration{
				{
					Name: to.StringPtr("default"),
					VirtualNetworkGatewayIPConfigurationPropertiesFormat: &network.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: network.Dynamic,
						PublicIPAddress: &network.SubResource{
							ID: to.StringPtr(resourceid.ResourceID(
								conf.SubscriptionID,
								conf.resourceGroup,
								"Microsoft.Network/publicIPAddresses",
								farmconst.GatewayIPAddressName,
							)),
						},
						Subnet: &network.SubResource{
							ID: to.StringPtr(resourceid.ResourceID(
								conf.SubscriptionID,
								conf.resourceGroup,
								"Microsoft.Network/virtualNetworks",
								armconst.VnetName,
							) + "/subnets/" + farmconst.VnetGatewaySubnetName),
						},
					},
				},
			},
		},
		Name:     to.StringPtr(farmconst.GatewayName),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworkGateways"),
		Location: to.StringPtr(conf.region),
	}
}

func newConfig(log *logrus.Entry) (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}

	return &c, nil
}

func main() {
	if err := run(context.Background(), logrus.NewEntry(logrus.StandardLogger())); err != nil {
		panic(err)
	}
}
