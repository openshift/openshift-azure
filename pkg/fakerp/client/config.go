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
	SubscriptionID      string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	TenantID            string `envconfig:"AZURE_TENANT_ID" required:"true"`
	ClientID            string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret        string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	AADClientID         string `envconfig:"AZURE_AAD_CLIENT_ID"`
	AADClientSecret     string `envconfig:"AZURE_AAD_CLIENT_SECRET"`
	AADGroupAdminsID    string `envconfig:"AZURE_AAD_GROUP_ADMINS_ID"`
	DeployVersion       string `envconfig:"DEPLOY_VERSION" required:"true"`
	RunningUnderTest    string `envconfig:"RUNNING_UNDER_TEST"`
	WorkspaceResourceID string `envconfig:"AZURE_WORKSPACE_ID"`

	Region        string
	Regions       string `envconfig:"AZURE_REGIONS" required:"true"`
	ResourceGroup string `envconfig:"RESOURCEGROUP" required:"true"`

	DNSDomain        string `envconfig:"DNS_DOMAIN" required:"true"`
	DNSResourceGroup string `envconfig:"DNS_RESOURCEGROUP" required:"true"`

	ResourceGroupTTL string `envconfig:"RESOURCEGROUP_TTL"`
	Manifest         string `envconfig:"MANIFEST"`
	NoWait           bool   `envconfig:"NO_WAIT"`
}

func NewConfig(log *logrus.Entry) (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}
	regions := strings.Split(c.Regions, ",")
	rand.Seed(time.Now().UTC().UnixNano())
	c.Region = regions[rand.Intn(len(regions))]
	if c.Region == "" {
		return nil, fmt.Errorf("must set AZURE_REGIONS to a comma separated list")
	}
	log.Infof("using region %s", c.Region)
	if c.WorkspaceResourceID == "" {
		c.WorkspaceResourceID = fmt.Sprintf(
			"/subscriptions/%s/resourcegroups/defaultresourcegroup-%s/providers/"+
				"microsoft.operationalinsights/workspaces/DefaultWorkspace-%s",
			c.SubscriptionID, c.Region, c.Region)
	}
	return &c, nil
}
