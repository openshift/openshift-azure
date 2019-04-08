package client

import (
	"fmt"
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
	Regions       string `envconfig:"AZURE_REGIONS"`
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
	// After v3/v4 die, goal here is to use AZURE_REGIONS both to define the
	// region set externally via secrets/secret and allow restricting it.  In
	// the interim, v3/v4 has a hard-coded region set which can be overridden by
	// AZURE_REGION; v5 (and presumably v4.1) will use AZURE_REGIONS.  Also
	// allow a cross-over period during which AZURE_REGIONS is not yet defined
	// in developer environments.
	if c.Regions != "" {
		regions := strings.Split(c.Regions, ",")
		rand.Seed(time.Now().UTC().UnixNano())
		c.Region = regions[rand.Intn(len(regions))]
	}
	if c.Region == "" {
		return nil, fmt.Errorf("must set AZURE_REGION and/or AZURE_REGIONS")
	}
	log.Infof("using region %s", c.Region)
	if c.AADClientID == "" {
		c.AADClientID = c.ClientID
		c.AADClientSecret = c.ClientSecret
	}
	return &c, nil
}
