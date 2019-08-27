package arm

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/util/arm"
	pl "github.com/openshift/openshift-azure/pkg/util/azureclient/network/privatelink"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

func privateLinkService(cs *api.OpenShiftManagedCluster) *pl.PrivateLinkService {
	subList := []string{"*"}
	return &pl.PrivateLinkService{
		PrivateLinkServiceProperties: &pl.PrivateLinkServiceProperties{
			Visibility: &pl.PrivateLinkServicePropertiesVisibility{
				Subscriptions: &subList,
			},
			AutoApproval: &pl.PrivateLinkServicePropertiesAutoApproval{
				Subscriptions: &subList,
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
					Name: to.StringPtr(arm.PrivateLinkNicName),
				},
			},
		},
		Name:     to.StringPtr(arm.PrivateLinkName),
		Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
		Version:  to.StringPtr("2019-06-01"),
		Location: to.StringPtr(cs.Location),
	}
}

func privateEndpoint(cs *api.OpenShiftManagedCluster, conf *client.Config, now int64) *pl.PrivateEndpoint {
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
							arm.PrivateLinkName)),
					},
				},
			},
			Subnet: &network.Subnet{
				ID: to.StringPtr(resourceid.ResourceID(
					cs.Properties.AzProfile.SubscriptionID,
					conf.ManagementResourceGroup,
					"Microsoft.Network/virtualNetworks",
					arm.VnetName,
				) + "/subnets/" + arm.VnetManagementSubnetName),
			},
		},
		Name:     to.StringPtr(fmt.Sprintf("%s-%s-%d", arm.PrivateEndpointNamePrefix, cs.Name, now)),
		Type:     to.StringPtr("Microsoft.Network/privateEndpoints"),
		Version:  to.StringPtr("2019-06-01"),
		Location: to.StringPtr(cs.Location),
		Tags:     tags,
	}
}
