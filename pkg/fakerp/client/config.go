package client

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/random"
)

var supportedRegions = []string{
	//"australiaeast",
	//"canadacentral",
	//"canadaeast",
	"eastus",
	//"westcentralus",
	"westeurope",
	//"westus",
}

type Config struct {
	SubscriptionID  string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	TenantID        string `envconfig:"AZURE_TENANT_ID" required:"true"`
	ClientID        string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret    string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	AADClientID     string `envconfig:"AZURE_AAD_CLIENT_ID"`
	AADClientSecret string `envconfig:"AZURE_AAD_CLIENT_SECRET"`

	Region        string `envconfig:"AZURE_REGION"`
	ResourceGroup string `envconfig:"RESOURCEGROUP"`

	NoGroupTags      bool   `envconfig:"NOGROUPTAGS"`
	ResourceGroupTTL string `envconfig:"RESOURCEGROUP_TTL"`
	Manifest         string `envconfig:"MANIFEST"`
	NoWait           bool   `envconfig:"NO_WAIT"`
}

func NewConfig(log *logrus.Entry, needRegion bool) (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}
	if needRegion {
		if c.Region == "" {
			// Randomly assign a supported region
			rand.Seed(time.Now().UTC().UnixNano())
			c.Region = supportedRegions[rand.Intn(len(supportedRegions))]
			log.Infof("using randomly selected region %s", c.Region)
			os.Setenv("AZURE_REGION", c.Region)
		}

		var supported bool
		for _, region := range supportedRegions {
			if c.Region == region {
				supported = true
			}
		}
		if !supported {
			return nil, fmt.Errorf("%s is not a supported region (supported regions: %v)", c.Region, supportedRegions)
		}
	}
	if c.ResourceGroup == "" {
		// Generate a resource group name
		suffix, err := random.LowerCaseAlphanumericString(8)
		if err != nil {
			return nil, err
		}
		c.ResourceGroup = fmt.Sprintf("generated-%s", suffix)
		log.Infof("using generated resource group name %s", c.ResourceGroup)
		os.Setenv("RESOURCEGROUP", c.ResourceGroup)
	}
	return &c, nil
}
