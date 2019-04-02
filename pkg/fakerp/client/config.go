package client

import (
	"math/rand"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

type Config struct {
	SubscriptionID   string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	TenantID         string `envconfig:"AZURE_TENANT_ID" required:"true"`
	ClientID         string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret     string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	AADClientID      string `envconfig:"AZURE_AAD_CLIENT_ID"`
	AADClientSecret  string `envconfig:"AZURE_AAD_CLIENT_SECRET"`
	AADGroupAdminsID string `envconfig:"AZURE_AAD_GROUP_ADMINS_ID"`
	DeployVersion    string `envconfig:"DEPLOY_VERSION" required:"true"`
	RunningUnderTest string `envconfig:"RUNNING_UNDER_TEST"`

	Region        string `envconfig:"AZURE_REGION"`
	ResourceGroup string `envconfig:"RESOURCEGROUP" required:"true"`

	ResourceGroupTTL string `envconfig:"RESOURCEGROUP_TTL"`
	Manifest         string `envconfig:"MANIFEST"`
	NoWait           bool   `envconfig:"NO_WAIT"`
}

func NewConfig(log *logrus.Entry) (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}
	if c.Region == "" {
		c.Region = "eastus" // remove in a follow-up and set Region required:"true"
	}
	regions := strings.Split(c.Region, ",")
	if len(regions) > 1 {
		rand.Seed(time.Now().UTC().UnixNano())
		c.Region = regions[rand.Intn(len(regions))]
	}
	log.Infof("using region %s", c.Region)
	if c.AADClientID == "" {
		c.AADClientID = c.ClientID
		c.AADClientSecret = c.ClientSecret
	}
	return &c, nil
}
