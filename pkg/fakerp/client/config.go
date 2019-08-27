package client

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
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

	ManagementResourceGroup string
	Region                  string `envconfig:"AZURE_REGION" required:"false"`
	Regions                 string `envconfig:"AZURE_REGIONS" required:"true"`
	ResourceGroup           string `envconfig:"RESOURCEGROUP" required:"true"`

	DNSDomain        string `envconfig:"DNS_DOMAIN" required:"true"`
	DNSResourceGroup string `envconfig:"DNS_RESOURCEGROUP" required:"true"`

	ResourceGroupTTL string `envconfig:"RESOURCEGROUP_TTL"`
	Manifest         string `envconfig:"MANIFEST"`
	NoWait           bool   `envconfig:"NO_WAIT"`
}

// NewConfig parses env variables and sets fakeRP configuration.
// This function is being re-used in client and server side of fakeRP.
// Server side uses CS object, passed from client in a form of manifest to
// prepopulate some of the random generated fields with the same value
// as client side.
func NewConfig(log *logrus.Entry, cs *api.OpenShiftManagedCluster) (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}

	// we need this, otherwise we sometimes end-up with client creating RG
	// in region A and server side creating all resources in region B
	if cs != nil && len(cs.Location) > 0 {
		c.Region = cs.Location
	} else {
		if len(c.Region) == 0 {
			regions := strings.Split(c.Regions, ",")
			rand.Seed(time.Now().UTC().UnixNano())
			c.Region = regions[rand.Intn(len(regions))]
			if c.Region == "" {
				return nil, fmt.Errorf("must set AZURE_REGIONS to a comma separated list")
			}
		} else if len(c.Region) > 0 && !strings.Contains(c.Regions, c.Region) {
			return nil, fmt.Errorf("must set AZURE_REGION to the one of the regions from AZURE_REGIONS")
		}
	}
	log.Infof("using region %s", c.Region)

	// Set management RG name
	c.ManagementResourceGroup = fmt.Sprintf("management-%s", c.Region)
	log.Infof("using management resource group %s", c.ManagementResourceGroup)

	return &c, nil
}
