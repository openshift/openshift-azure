package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/installer/pkg/terraform/exec/plugins"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/asset/store"
	targetassets "github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/azure"
	"github.com/openshift/installer/pkg/types/defaults"
	openstackvalidation "github.com/openshift/installer/pkg/types/openstack/validation"
	"github.com/openshift/installer/pkg/types/validation"
	pkgvalidate "github.com/openshift/installer/pkg/validate"
	"github.com/openshift/openshift-azure/pkg/util/random"
)

var (
	gitCommit  = "unknown"
	baseDomain = "osadev.cloud"
)

type EnvConfig struct {
	SubscriptionID   string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	ClientID         string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret     string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	TenantID         string `envconfig:"AZURE_TENANT_ID" required:"true"`
	DNSResourceGroup string `envconfig:"DNS_RESOURCEGROUP" required:"true"`

	SSHKey string `envconfig:"SSH_KEY" required:"true"`

	Region  string
	Regions string `envconfig:"AZURE_REGIONS" required:"true"`
}

func main() {

	if len(os.Args) > 0 {
		base := filepath.Base(os.Args[0])
		cname := strings.TrimSuffix(base, filepath.Ext(base))
		if pluginRunner, ok := plugins.KnownPlugins[cname]; ok {
			pluginRunner()
			return
		}
	}

	if err := run(); err != nil {
		panic(err)
	}
}

func newEnvConfig() (*EnvConfig, error) {
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
	return &c, nil
}

func readSSHKey(path string) (string, error) {
	keyAsBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	key := string(keyAsBytes)

	err = pkgvalidate.SSHPublicKey(key)
	if err != nil {
		return "", err
	}

	return key, nil
}

func getInstallConfig(name string, ec *EnvConfig) (*types.InstallConfig, error) {
	// TODO: move to util/secrets
	fqdn, err := random.FQDN(baseDomain, 5)
	if err != nil {
		return nil, err
	}

	file, err := os.Open("secrets/pull-secret.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	pullSecret, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	sshKey, err := readSSHKey(ec.SSHKey)
	if err != nil {
		return nil, err
	}
	cfg := types.InstallConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: types.InstallConfigVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		BaseDomain: fqdn,
		Compute: []types.MachinePool{
			{
				Name:           "worker",
				Replicas:       to.Int64Ptr(1),
				Hyperthreading: types.HyperthreadingEnabled,
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
			Azure: &azure.Platform{
				Region:                      ec.Region,
				BaseDomainResourceGroupName: ec.DNSResourceGroup,
			},
		},
		PullSecret: string(pullSecret),
		SSHKey:     sshKey,
	}
	return &cfg, nil
}

func saveCredentials(credentials icazure.Credentials, filePath string) error {
	jsonCreds, err := json.Marshal(credentials)
	err = os.MkdirAll(filepath.Dir(filePath), 0700)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, jsonCreds, 0600)
}

func run() error {
	rootCmd := &cobra.Command{
		Use:  "./azure [component]",
		Long: "Azure Red Hat OpenShift dispatcher",
	}
	rootCmd.PersistentFlags().StringP("loglevel", "l", "Debug", "Valid values are [Debug, Info, Warning, Error]")
	rootCmd.PersistentFlags().StringP("name", "n", "demo-cluster", "Cluster name value")
	rootCmd.Printf("gitCommit %s\n", gitCommit)
	rootCmd.Execute()

	name, err := rootCmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	ec, err := newEnvConfig()
	if err != nil {
		return err
	}
	cfg, err := getInstallConfig(name, ec)
	if err != nil {
		return errors.Wrap(err, "failed to get InstallConfig")
	}

	err = saveCredentials(icazure.Credentials{
		SubscriptionID: ec.SubscriptionID,
		ClientID:       ec.ClientID,
		ClientSecret:   ec.ClientSecret,
		TenantID:       ec.TenantID,
	}, filepath.Join(os.Getenv("HOME"), ".azure", "osServicePrincipal.json"))
	if err != nil {
		return err
	}

	defaults.SetInstallConfigDefaults(cfg)
	if err := validation.ValidateInstallConfig(cfg, openstackvalidation.NewValidValuesFetcher()).ToAggregate(); err != nil {
		return errors.Wrap(err, "invalid install config")
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal InstallConfig")
	}

	// doing this to prevent the stdin questions.
	ic := &installconfig.InstallConfig{}
	ic.Config = cfg
	ic.File = &asset.File{
		Filename: "install-config.yaml",
		Data:     data,
	}
	directory := path.Join("clusters", name)
	assetStore, err := store.NewStore(directory)
	if err != nil {
		return errors.Wrap(err, "failed to create asset store")
	}
	if err := asset.PersistToFile(ic, directory); err != nil {
		return errors.Wrap(err, "failed to write install config")
	}

	targets := targetassets.InstallConfig
	targets = append(targets, targetassets.IgnitionConfigs...)
	targets = append(targets, targetassets.Manifests...)
	targets = append(targets, targetassets.Cluster...)

	for _, a := range targets {
		err := assetStore.Fetch(a, targets...)
		if err != nil {
			err = errors.Wrapf(err, "failed to fetch %s", a.Name())
		}

		if err2 := asset.PersistToFile(a, directory); err2 != nil {
			err2 = errors.Wrapf(err2, "failed to write asset (%s) to disk", a.Name())
			if err != nil {
				return err
			}
			return err2
		}

		if err != nil {
			return err
		}
	}

	// wait for the cluster to come up
	//	err = waitForBootstrapComplete(ctx, config, rootOpts.dir)
	//	err = destroybootstrap.Destroy(rootOpts.dir)
	//	err = waitForInstallComplete(ctx, config, rootOpts.dir)

	return nil
}
