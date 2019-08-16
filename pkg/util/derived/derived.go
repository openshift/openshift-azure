package derived

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
)

func baseCloudProviderConf(cs *api.OpenShiftManagedCluster, disableOutboundSNAT bool) (*cloudprovider.Config, error) {
	cfg := cloudprovider.Config{
		TenantID:                     cs.Properties.AzProfile.TenantID,
		SubscriptionID:               cs.Properties.AzProfile.SubscriptionID,
		ResourceGroup:                cs.Properties.AzProfile.ResourceGroup,
		LoadBalancerSku:              "standard",
		Location:                     cs.Location,
		SecurityGroupName:            "nsg-worker",
		VMType:                       "vmss",
		SubnetName:                   "default",
		VnetName:                     "vnet",
		UseInstanceMetadata:          true,
		CloudProviderBackoff:         true,
		CloudProviderBackoffRetries:  6,
		CloudProviderBackoffJitter:   1.0,
		CloudProviderBackoffDuration: 5,
		CloudProviderBackoffExponent: 1.5,
		CloudProviderRateLimit:       true,
		CloudProviderRateLimitQPS:    3.0,
		CloudProviderRateLimitBucket: 10,
	}
	if disableOutboundSNAT {
		cfg.DisableOutboundSNAT = to.BoolPtr(disableOutboundSNAT)
	}
	return &cfg, nil
}

// MasterCloudProviderConf returns cloudprovider config for masters
func MasterCloudProviderConf(cs *api.OpenShiftManagedCluster, disableOutboundSNAT bool) ([]byte, error) {
	cpc, err := baseCloudProviderConf(cs, disableOutboundSNAT)
	if err != nil {
		return nil, err
	}
	cpc.AadClientID = cs.Properties.MasterServicePrincipalProfile.ClientID
	cpc.AadClientSecret = cs.Properties.MasterServicePrincipalProfile.Secret
	return yaml.Marshal(cpc)
}

// WorkerCloudProviderConf returns cloudprovider config for workers
func WorkerCloudProviderConf(cs *api.OpenShiftManagedCluster, disableOutboundSNAT bool) ([]byte, error) {
	cpc, err := baseCloudProviderConf(cs, disableOutboundSNAT)
	if err != nil {
		return nil, err
	}
	cpc.AadClientID = cs.Properties.WorkerServicePrincipalProfile.ClientID
	cpc.AadClientSecret = cs.Properties.WorkerServicePrincipalProfile.Secret
	return yaml.Marshal(cpc)
}

func AadGroupSyncConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	provider := cs.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider)
	return yaml.Marshal(provider)
}
