package derived

import (
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	armconst "github.com/openshift/openshift-azure/pkg/arm/constants"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/pluginversion"
)

func baseCloudProviderConf(cs *api.OpenShiftManagedCluster) (*cloudprovider.Config, error) {
	cfg := cloudprovider.Config{
		TenantID:                     cs.Properties.AzProfile.TenantID,
		SubscriptionID:               cs.Properties.AzProfile.SubscriptionID,
		ResourceGroup:                cs.Properties.AzProfile.ResourceGroup,
		LoadBalancerSku:              armconst.LoadBalancerSku,
		Location:                     cs.Location,
		SecurityGroupName:            armconst.NsgWorkerName,
		VMType:                       armconst.VmssType,
		SubnetName:                   armconst.VnetSubnetName,
		VnetName:                     armconst.VnetName,
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
	major, _, _ := pluginversion.Parse(cs.Config.PluginVersion)
	if major >= 15 {
		cfg.CloudProviderRateLimitQPS = 10.0
		cfg.CloudProviderRateLimitBucket = 100
	}
	return &cfg, nil
}

// MasterCloudProviderConf returns cloudprovider config for masters
func MasterCloudProviderConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	cpc, err := baseCloudProviderConf(cs)
	if err != nil {
		return nil, err
	}
	cpc.AadClientID = cs.Properties.MasterServicePrincipalProfile.ClientID
	cpc.AadClientSecret = cs.Properties.MasterServicePrincipalProfile.Secret
	return yaml.Marshal(cpc)
}

// WorkerCloudProviderConf returns cloudprovider config for workers
func WorkerCloudProviderConf(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	cpc, err := baseCloudProviderConf(cs)
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
