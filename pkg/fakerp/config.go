package fakerp

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

var supportedRegions = []string{
	"australiasoutheast",
	"eastus",
	"westeurope",
}

type Config struct {
	Username        string `envconfig:"AZURE_USERNAME"`
	Password        string `envconfig:"AZURE_PASSWORD"`
	SubscriptionID  string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	TenantID        string `envconfig:"AZURE_TENANT_ID" required:"true"`
	ClientID        string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret    string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	AADClientID     string `envconfig:"AZURE_AAD_CLIENT_ID"`
	AADClientSecret string `envconfig:"AZURE_AAD_CLIENT_SECRET"`

	Region           string `envconfig:"AZURE_REGION"`
	DnsDomain        string `envconfig:"DNS_DOMAIN" required:"true"`
	DnsResourceGroup string `envconfig:"DNS_RESOURCEGROUP" required:"true"`
	ResourceGroup    string `envconfig:"RESOURCEGROUP" required:"true"`

	NoGroupTags      bool   `envconfig:"NOGROUPTAGS"`
	ResourceGroupTTL string `envconfig:"RESOURCEGROUP_TTL"`
	Manifest         string `envconfig:"MANIFEST" default:"test/manifests/normal/create.yaml"`
}

func NewConfig() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}

	if c.Region == "" {
		// Randomly assign a supported region
		rand.Seed(time.Now().UTC().UnixNano())
		c.Region = supportedRegions[rand.Intn(len(supportedRegions))]
		logrus.Infof("using randomly selected region %q", c.Region)
	}

	var supported bool
	for _, region := range supportedRegions {
		if c.Region == region {
			supported = true
		}
	}
	if !supported {
		return nil, fmt.Errorf("%q is not a supported region (supported regions: %v)", c.Region, supportedRegions)
	}
	return &c, nil
}
