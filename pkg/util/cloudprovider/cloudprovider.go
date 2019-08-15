package cloudprovider

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

// See upstream Config for reference: https://github.com/openshift/origin/blob/release-3.11/vendor/k8s.io/kubernetes/pkg/cloudprovider/providers/azure/azure.go
type Config struct {
	TenantID                     string  `json:"tenantId,omitempty"`
	SubscriptionID               string  `json:"subscriptionId,omitempty"`
	AadClientID                  string  `json:"aadClientId,omitempty"`
	AadClientSecret              string  `json:"aadClientSecret,omitempty"`
	ResourceGroup                string  `json:"resourceGroup,omitempty"`
	Location                     string  `json:"location,omitempty"`
	LoadBalancerSku              string  `json:"loadBalancerSku,omitempty"`
	SecurityGroupName            string  `json:"securityGroupName,omitempty"`
	VMType                       string  `json:"vmType,omitempty"`
	SubnetName                   string  `json:"subnetName,omitempty"`
	VnetName                     string  `json:"vnetName,omitempty"`
	UseInstanceMetadata          bool    `json:"useInstanceMetadata,omitempty"`
	CloudProviderBackoff         bool    `json:"cloudProviderBackoff,omitempty"`
	CloudProviderBackoffDuration int     `json:"cloudProviderBackoffDuration,omitempty"`
	CloudProviderBackoffExponent float64 `json:"cloudProviderBackoffExponent,omitempty"`
	CloudProviderBackoffJitter   float64 `json:"cloudProviderBackoffJitter,omitempty"`
	CloudProviderBackoffRetries  int     `json:"cloudProviderBackoffRetries,omitempty"`
	CloudProviderRateLimit       bool    `json:"cloudProviderRateLimit,omitempty"`
	CloudProviderRateLimitBucket int     `json:"cloudProviderRateLimitBucket,omitempty"`
	CloudProviderRateLimitQPS    float32 `json:"cloudProviderRateLimitQPS,omitempty"`
	// DisableOutboundSNAT disables the outbound SNAT for public load balancer rules.
	// It should only be set when loadBalancerSku is standard. If not set, it will be default to false.
	DisableOutboundSNAT *bool `json:"disableOutboundSNAT,omitempty"`
}

// Load returns Config unmarshalled from the file provided
func Load(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read "+path)
	}

	var m Config
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal "+path)
	}
	return &m, nil
}
