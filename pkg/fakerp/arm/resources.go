package arm

import (
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	armconst "github.com/openshift/openshift-azure/pkg/arm/constants"
	farmconst "github.com/openshift/openshift-azure/pkg/fakerp/arm/constants"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

var (
	// The versions referenced here must be kept in lockstep with the imports above.
	versionMap = map[string]string{
		"Microsoft.Network": "2019-06-01",
	}
)

func privateLinkService(cs *api.OpenShiftManagedCluster) *arm.Resource {
	return &arm.Resource{
		Resource: &network.PrivateLinkService{
			PrivateLinkServiceProperties: &network.PrivateLinkServiceProperties{
				Visibility: &network.PrivateLinkServicePropertiesVisibility{
					Subscriptions: &[]string{os.Getenv("AZURE_SUBSCRIPTION_ID")},
				},
				AutoApproval: &network.PrivateLinkServicePropertiesAutoApproval{
					Subscriptions: &[]string{os.Getenv("AZURE_SUBSCRIPTION_ID")},
				},
				LoadBalancerFrontendIPConfigurations: &[]network.FrontendIPConfiguration{
					{
						ID: to.StringPtr(cs.Properties.NetworkProfile.InternalLoadBalancerFrontendIPID),
					},
				},
				IPConfigurations: &[]network.PrivateLinkServiceIPConfiguration{
					{
						PrivateLinkServiceIPConfigurationProperties: &network.PrivateLinkServiceIPConfigurationProperties{
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
			Location: to.StringPtr(cs.Location),
		},
		APIVersion: versionMap["Microsoft.Network"],
	}
}

func privateEndpoint(cs *api.OpenShiftManagedCluster, conf *client.Config) *arm.Resource {
	tags := map[string]*string{
		"now": to.StringPtr(fmt.Sprintf("%d", time.Now().Unix())),
		"ttl": to.StringPtr("72h"),
	}
	if conf.ResourceGroupTTL != "" {
		tags["ttl"] = &conf.ResourceGroupTTL
	}
	return &arm.Resource{
		Resource: &network.PrivateEndpoint{
			PrivateEndpointProperties: &network.PrivateEndpointProperties{
				ManualPrivateLinkServiceConnections: &[]network.PrivateLinkServiceConnection{
					{
						Name: to.StringPtr("plsConnection"),
						PrivateLinkServiceConnectionProperties: &network.PrivateLinkServiceConnectionProperties{
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
			Location: to.StringPtr(cs.Location),
			Tags:     tags,
		},
		APIVersion: versionMap["Microsoft.Network"],
	}
}
