package cloudprovider

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

type Config struct {
	TenantID            string `json:"tenantId,omitempty"`
	SubscriptionID      string `json:"subscriptionId,omitempty"`
	AadClientID         string `json:"aadClientId,omitempty"`
	AadClientSecret     string `json:"aadClientSecret,omitempty"`
	ResourceGroup       string `json:"resourceGroup,omitempty"`
	Location            string `json:"location,omitempty"`
	LoadBalancerSku     string `json:"loadBalancerSku,omitempty"`
	SecurityGroupName   string `json:"securityGroupName,omitempty"`
	VMType              string `json:"vmType,omitempty"`
	SubnetName          string `json:"subnetName,omitempty"`
	VnetName            string `json:"vnetName,omitempty"`
	UseInstanceMetadata bool   `json:"useInstanceMetadata,omitempty"`
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
