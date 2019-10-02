package arm

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	armconst "github.com/openshift/openshift-azure/pkg/arm/constants"
	farmconst "github.com/openshift/openshift-azure/pkg/fakerp/arm/constants"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	pl "github.com/openshift/openshift-azure/pkg/util/azureclient/network/privatelink"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

func privateLinkService(cs *api.OpenShiftManagedCluster) *pl.PrivateLinkService {
	return &pl.PrivateLinkService{
		PrivateLinkServiceProperties: &pl.PrivateLinkServiceProperties{
			Visibility: &pl.PrivateLinkServicePropertiesVisibility{
				Subscriptions: &[]string{"*"},
			},
			AutoApproval: &pl.PrivateLinkServicePropertiesAutoApproval{
				Subscriptions: &[]string{"*"},
			},
			LoadBalancerFrontendIPConfigurations: &[]network.FrontendIPConfiguration{
				{
					ID: to.StringPtr(cs.Properties.NetworkProfile.InternalLoadBalancerFrontendIPID),
				},
			},
			IPConfigurations: &[]pl.PrivateLinkServiceIPConfiguration{
				{
					PrivateLinkServiceIPConfigurationProperties: &pl.PrivateLinkServiceIPConfigurationProperties{
						PrivateIPAllocationMethod: network.Dynamic,
						Subnet: &network.Subnet{
							ID: to.StringPtr(cs.Properties.NetworkProfile.ManagementSubnetID),
						},
					},
					Name: to.StringPtr(farmconst.PrivateLinkNicName),
				},
			},
		},
		Name:     to.StringPtr(farmconst.PrivateLinkName),
		Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
		Version:  to.StringPtr("2019-06-01"),
		Location: to.StringPtr(cs.Location),
	}
}

func privateEndpoint(cs *api.OpenShiftManagedCluster, conf *client.Config) *pl.PrivateEndpoint {
	tags := map[string]*string{
		"now": to.StringPtr(fmt.Sprintf("%d", time.Now().Unix())),
		"ttl": to.StringPtr("72h"),
	}
	if conf.ResourceGroupTTL != "" {
		tags["ttl"] = &conf.ResourceGroupTTL
	}
	return &pl.PrivateEndpoint{
		PrivateEndpointProperties: &pl.PrivateEndpointProperties{
			ManualPrivateLinkServiceConnections: &[]pl.PrivateLinkServiceConnection{
				{
					Name: to.StringPtr("plsConnection"),
					PrivateLinkServiceConnectionProperties: &pl.PrivateLinkServiceConnectionProperties{
						PrivateLinkServiceID: to.StringPtr(resourceid.ResourceID(
							cs.Properties.AzProfile.SubscriptionID,
							cs.Properties.AzProfile.ResourceGroup,
							"Microsoft.Network/privateLinkServices",
							farmconst.PrivateLinkName)),
					},
				},
			},
			Subnet: &network.Subnet{
				ID: to.StringPtr(resourceid.ResourceID(
					cs.Properties.AzProfile.SubscriptionID,
					conf.ManagementResourceGroup,
					"Microsoft.Network/virtualNetworks",
					armconst.VnetName,
				) + "/subnets/" + armconst.VnetManagementSubnetName),
			},
		},
		Name:     to.StringPtr(fmt.Sprintf("%s-%s", farmconst.PrivateEndpointNamePrefix, cs.Name)),
		Type:     to.StringPtr("Microsoft.Network/privateEndpoints"),
		Version:  to.StringPtr("2019-06-01"),
		Location: to.StringPtr(cs.Location),
		Tags:     tags,
	}
}
