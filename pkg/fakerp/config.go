package fakerp

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/tls"
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
	NoWait           bool   `envconfig:"NO_WAIT"`
}

const (
	DataDirectory           = "_data/"
	LoggingSecretsDirectory = "secrets/"
)

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
	}
	return &c, nil
}

func GetPluginConfig() (*api.PluginConfig, error) {
	tc := api.TestConfig{
		RunningUnderTest:      os.Getenv("RUNNING_UNDER_TEST") == "true",
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
	artifactDir, err := FindDirectory(LoggingSecretsDirectory)
	if err != nil {
		return nil, err
	}
	logCert, err := readCert(filepath.Join(artifactDir, "logging-int.cert"))
	if err != nil {
		return nil, err
	}
	logKey, err := readKey(filepath.Join(artifactDir, "logging-int.key"))
	if err != nil {
		return nil, err
	}
	metCert, err := readCert(filepath.Join(artifactDir, "metrics-int.cert"))
	if err != nil {
		return nil, err
	}
	metKey, err := readKey(filepath.Join(artifactDir, "metrics-int.key"))
	if err != nil {
		return nil, err
	}
	pullSecret, err := readFile(filepath.Join(artifactDir, ".dockerconfigjson"))
	if err != nil {
		return nil, err
	}

	var syncImage, metricsBridge string
	if os.Getenv("SYNC_IMAGE") == "" {
		syncImage = "quay.io/openshift-on-azure/sync:latest"
	} else {
		syncImage = os.Getenv("SYNC_IMAGE")
	}
	if os.Getenv("METRICSBRIDGE_IMAGE") == "" {
		metricsBridge = "quay.io/openshift-on-azure/metricsbridge:latest"
	} else {
		metricsBridge = os.Getenv("METRICSBRIDGE_IMAGE")
	}

	genevaConfig := api.GenevaConfig{
		ImagePullSecret: pullSecret,

		LoggingCert:                logCert,
		LoggingKey:                 logKey,
		LoggingSector:              "US-Test",
		LoggingAccount:             "ccpopenshiftdiag",
		LoggingNamespace:           "CCPOpenShift",
		LoggingControlPlaneAccount: "RPOpenShiftAccount",
		LoggingImage:               "osarpint.azurecr.io/acs/mdsd:12051806",
		TDAgentImage:               "osarpint.azurecr.io/acs/td-agent:latest",

		MetricsCert:     metCert,
		MetricsKey:      metKey,
		MetricsBridge:   metricsBridge,
		StatsdImage:     "osarpint.azurecr.io/acs/mdm:git-a909a2e76",
		MetricsAccount:  "RPOpenShift",
		MetricsEndpoint: "https://az-int.metrics.nsatc.net/",
	}
	return &api.PluginConfig{
		SyncImage:       syncImage,
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
		GenevaConfig:    genevaConfig,
	}, nil
}

func readCert(path string) (*x509.Certificate, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParseCert(b)
}

func readKey(path string) (*rsa.PrivateKey, error) {
	b, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return tls.ParsePrivateKey(b)
}

func readFile(path string) ([]byte, error) {
	if fileExist(path) {
		return ioutil.ReadFile(path)
	}
	return []byte{}, fmt.Errorf("file %s does not exist", path)
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
