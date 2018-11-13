package fakerp

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/openshift/openshift-azure/pkg/tls"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
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
	Manifest         string `envconfig:"MANIFEST"`
}

func NewConfig(log *logrus.Entry) (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}

	if c.Manifest == "" {
		c.Manifest = "test/manifests/normal/create.yaml"
	}

	if c.Region == "" {
		// Randomly assign a supported region
		rand.Seed(time.Now().UTC().UnixNano())
		c.Region = supportedRegions[rand.Intn(len(supportedRegions))]
		log.Infof("using randomly selected region %q", c.Region)
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
	os.Setenv("AZURE_REGION", c.Region)
	return &c, nil
}

func getPluginConfig() (*api.PluginConfig, error) {
	tc := api.TestConfig{
		RunningUnderTest:      os.Getenv("RUNNING_UNDER_TEST") != "",
		ImageResourceGroup:    os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:     os.Getenv("IMAGE_RESOURCENAME"),
		DeployOS:              os.Getenv("DEPLOY_OS"),
		ImageOffer:            os.Getenv("IMAGE_OFFER"),
		ImageVersion:          os.Getenv("IMAGE_VERSION"),
		ORegURL:               os.Getenv("OREG_URL"),
		EtcdBackupImage:       os.Getenv("ETCDBACKUP_IMAGE"),
		AzureControllersImage: os.Getenv("AZURE_CONTROLLERS_IMAGE"),
	}

	// populate geneva artifacts
	var cb, kb, db []byte
	var crt *x509.Certificate
	var key *rsa.PrivateKey
	var err error
	var syncImage string
	artifactDir := "./secrets/azure/"
	// if test files exist, load them
	if fileExist(artifactDir+"logging-int.cert") &&
		fileExist(artifactDir+"logging-int.key") &&
		fileExist(artifactDir+".dockerconfigjson") {
		cb, err = ioutil.ReadFile(artifactDir + "logging-int.cert")
		if err != nil {
			return nil, err
		}
		kb, err = ioutil.ReadFile(artifactDir + "logging-int.key")
		if err != nil {
			return nil, err
		}
		crt, err = tls.ParseCert(cb)
		if err != nil {
			return nil, err
		}
		key, err = tls.ParsePrivateKey(kb)
		if err != nil {
			return nil, err
		}
		db, err = ioutil.ReadFile(artifactDir + ".dockerconfigjson")
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("azure secrets files does not exist in ./secrets/azure")
	}

	if os.Getenv("SYNC_IMAGE") == "" {
		syncImage = "quay.io/openshift-on-azure/sync:latest"
	} else {
		syncImage = os.Getenv("SYNC_IMAGE")
	}

	genevaConfig := api.GenevaConfig{
		LoggingCert:     crt,
		LoggingKey:      key,
		ImagePullSecret: db,
	}

	return &api.PluginConfig{
		SyncImage:       syncImage,
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
		GenevaConfig:    genevaConfig,
	}, nil
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
