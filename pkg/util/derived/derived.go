package derived

import (
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
)

func baseCloudProviderConf(cs *api.OpenShiftManagedCluster, useInstanceMetadata bool, cloudProviderBackoff bool) *cloudprovider.Config {
	cfg := cloudprovider.Config{
		TenantID:             cs.Properties.AzProfile.TenantID,
		SubscriptionID:       cs.Properties.AzProfile.SubscriptionID,
		ResourceGroup:        cs.Properties.AzProfile.ResourceGroup,
		LoadBalancerSku:      "standard",
		Location:             cs.Location,
		SecurityGroupName:    "nsg-worker",
		VMType:               "vmss",
		SubnetName:           "default",
		VnetName:             "vnet",
		UseInstanceMetadata:  useInstanceMetadata, // TODO: hard-wire to true after v3 has gone
		CloudProviderBackoff: cloudProviderBackoff,
	}
	if cloudProviderBackoff {
		cfg.CloudProviderBackoffRetries = 6
		cfg.CloudProviderBackoffJitter = 1.0
		cfg.CloudProviderBackoffDuration = 5
		cfg.CloudProviderBackoffExponent = 1.5
		cfg.CloudProviderRateLimit = cloudProviderBackoff
		cfg.CloudProviderRateLimitQPS = 3.0
		cfg.CloudProviderRateLimitBucket = 10
	}
	return &cfg
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
