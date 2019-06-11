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

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/kelseyhightower/envconfig"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

// GetInstallConfig returns pre-populated install config
func GetInstallConfig(name string, ec *EnvConfig) (*types.InstallConfig, error) {
	file, err := os.Open("secrets/pull-secret.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	pullSecret, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var key *rsa.PrivateKey
	var pubKey string
	if os.Getenv("SSH_KEY") == "" {
		var err error
		if key, err = tls.NewPrivateKey(); err != nil {
			return nil, err
		}
		pubKey, err = tls.SSHPublicKeyAsString(&key.PublicKey)
		if err != nil {
			return nil, err
		}
		err = writeToFile(pubKey, filepath.Join(ec.Directory, "id_rsa.pub"))
		if err != nil {
			return nil, err
		}
		b, err := tls.PrivateKeyAsBytes(key)
		if err != nil {
			return nil, err
		}
		if writeToFile(string(b), filepath.Join(ec.Directory, "id_rsa")) != nil {
			return nil, err
		}
	} else {
		pubKey, err = readFile(os.Getenv("SSH_KEY"))
		if err != nil {
			return nil, err
		}
	}

	cfg := types.InstallConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: types.InstallConfigVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		BaseDomain: baseDomain,
		Compute: []types.MachinePool{
			{
				Name:           "worker",
				Replicas:       to.Int64Ptr(3),
				Hyperthreading: types.HyperthreadingEnabled,
				Platform: types.MachinePoolPlatform{
					Azure: &azuretypes.MachinePool{
						Zones:        []string{"1", "2", "3"},
						InstanceType: "Standard_D3_v2",
					},
				},
			},
		},
		Networking: &types.Networking{
			MachineCIDR:    ipnet.MustParseCIDR("10.0.0.0/16"),
			NetworkType:    "OpenShiftSDN",
			ServiceNetwork: []ipnet.IPNet{*ipnet.MustParseCIDR("172.30.0.0/16")},
			ClusterNetwork: []types.ClusterNetworkEntry{
				{
					CIDR:       *ipnet.MustParseCIDR("10.128.0.0/14"),
					HostPrefix: 23,
				},
			},
		},
		ControlPlane: &types.MachinePool{
			Name:           "master",
			Replicas:       to.Int64Ptr(3),
			Hyperthreading: types.HyperthreadingEnabled,
		},
		Platform: types.Platform{
			Azure: &azuretypes.Platform{
				Region:                      ec.Region,
				BaseDomainResourceGroupName: ec.DNSResourceGroup,
			},
		},
		PullSecret: string(pullSecret),
		SSHKey:     pubKey,
	}
	return &cfg, nil
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
