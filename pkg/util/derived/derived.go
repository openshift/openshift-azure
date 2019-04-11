package derived

import (
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
)

func baseCloudProviderConf(cs *api.OpenShiftManagedCluster, useInstanceMetadata bool, cloudProviderBackoff bool) *cloudprovider.Config {
	return &cloudprovider.Config{
		TenantID:                     cs.Properties.AzProfile.TenantID,
		SubscriptionID:               cs.Properties.AzProfile.SubscriptionID,
		ResourceGroup:                cs.Properties.AzProfile.ResourceGroup,
		LoadBalancerSku:              "standard",
		Location:                     cs.Location,
		SecurityGroupName:            "nsg-worker",
		VMType:                       "vmss",
		SubnetName:                   "default",
		VnetName:                     "vnet",
		UseInstanceMetadata:          useInstanceMetadata, // TODO: hard-wire to true after v3 has gone
		CloudProviderBackoff:         cloudProviderBackoff,
		CloudProviderBackoffRetries:  6, // values are ignored if backoff is set to false
		CloudProviderBackoffJitter:   1.0,
		CloudProviderBackoffDuration: 6,
		CloudProviderBackoffExponent: 1.5,
		CloudProviderRateLimit:       cloudProviderBackoff,
		CloudProviderRateLimitQPS:    3.0, // values are ignored if ratelimit is set to false
		CloudProviderRateLimitBucket: 10,
	}
}

func MasterCloudProviderConf(cs *api.OpenShiftManagedCluster, useInstanceMetadata bool, cloudProviderBackoff bool) ([]byte, error) {
	cpc := baseCloudProviderConf(cs, useInstanceMetadata, cloudProviderBackoff)
	cpc.AadClientID = cs.Properties.MasterServicePrincipalProfile.ClientID
	cpc.AadClientSecret = cs.Properties.MasterServicePrincipalProfile.Secret
	return yaml.Marshal(cpc)
}

func WorkerCloudProviderConf(cs *api.OpenShiftManagedCluster, useInstanceMetadata bool, cloudProviderBackoff bool) ([]byte, error) {
	cpc := baseCloudProviderConf(cs, useInstanceMetadata, cloudProviderBackoff)
	cpc.AadClientID = cs.Properties.WorkerServicePrincipalProfile.ClientID
	cpc.AadClientSecret = cs.Properties.WorkerServicePrincipalProfile.Secret
	return yaml.Marshal(cpc)
}

func AadGroupSyncConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	provider := cs.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider)
	return yaml.Marshal(provider)
}
