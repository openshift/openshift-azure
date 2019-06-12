package config

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"

	"github.com/openshift/openshift-azure/pkg/util/tls"
)

var (
	baseDomain = "osadev.cloud"
)

// EnvConfig is fakeRP configuration struct
type EnvConfig struct {
	SubscriptionID   string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	ClientID         string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret     string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	TenantID         string `envconfig:"AZURE_TENANT_ID" required:"true"`
	DNSResourceGroup string `envconfig:"DNS_RESOURCEGROUP" required:"true"`

	SSHKey string `envconfig:"SSH_KEY" required:"false"`

	Region    string
	Regions   string `envconfig:"AZURE_REGIONS" required:"true"`
	Directory string `envconfig:"AZURE_DIRECTORY" required:"false"`
}

// NewEnvConfig return fakeRP configuration from env
func NewEnvConfig(name string) (*EnvConfig, error) {
	var c EnvConfig
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}
	regions := strings.Split(c.Regions, ",")
	rand.Seed(time.Now().UTC().UnixNano())
	c.Region = regions[rand.Intn(len(regions))]
	if c.Region == "" {
		return nil, fmt.Errorf("must set AZURE_REGIONS to a comma separated list")
	}
	if c.Directory == "" {
		// temporary staging folder
		directory := filepath.Join("_data/clusters", name)
		c.Directory = directory
		err := os.MkdirAll(directory, 0755)
		if err != nil {
			return nil, err
		}
	}
	return &c, nil
}

// EnrichInstallConfig returns pre-populated install config
func EnrichInstallConfig(name string, ec *EnvConfig, cfg *types.InstallConfig) error {
	file, err := os.Open("secrets/pull-secret.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	pullSecret, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	var key *rsa.PrivateKey
	var pubKey string
	if os.Getenv("SSH_KEY") == "" {
		var err error
		if key, err = tls.NewPrivateKey(); err != nil {
			return err
		}
		pubKey, err = tls.SSHPublicKeyAsString(&key.PublicKey)
		if err != nil {
			return err
		}
		err = writeToFile(pubKey, filepath.Join(ec.Directory, "id_rsa.pub"))
		if err != nil {
			return err
		}
		b, err := tls.PrivateKeyAsBytes(key)
		if err != nil {
			return err
		}
		if writeToFile(string(b), filepath.Join(ec.Directory, "id_rsa")) != nil {
			return err
		}
	} else {
		pubKey, err = readFile(os.Getenv("SSH_KEY"))
		if err != nil {
			return err
		}
	}

	cfg.Platform = types.Platform{
		Azure: &azuretypes.Platform{
			Region:                      ec.Region,
			BaseDomainResourceGroupName: ec.DNSResourceGroup,
		},
	}
	cfg.BaseDomain = baseDomain
	cfg.PullSecret = string(pullSecret)
	cfg.SSHKey = pubKey

	return nil
}

func writeToFile(data, saveFileTo string) error {
	err := ioutil.WriteFile(saveFileTo, []byte(data), 0600)
	if err != nil {
		return err
	}

	log.Printf("Data saved to: %s", saveFileTo)
	return nil
}

func readFile(path string) (string, error) {
	keyAsBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(keyAsBytes), nil
}
