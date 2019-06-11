package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/asset/store"
	targetassets "github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/terraform/exec/plugins"
	"github.com/openshift/installer/pkg/types/defaults"
	openstackvalidation "github.com/openshift/installer/pkg/types/openstack/validation"
	"github.com/openshift/installer/pkg/types/validation"

	fakerpconfig "github.com/openshift/openshift-azure/pkg/fakerp/config"
)

var (
	gitCommit = "unknown"
)

func main() {
	// TODO: This should move to installer /pkg
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

	// env variable configuration
	ec, err := fakerpconfig.NewEnvConfig(name)
	if err != nil {
		return err
	}

	// assetStore is responsible for all assets execution
	// TODO: directory and assetStore should get new SQL DB based implementation
	assetStore, err := store.NewStore(ec.Directory)
	if err != nil {
		return errors.Wrap(err, "failed to create asset store")
	}

	// install config
	cfg, err := fakerpconfig.GetInstallConfig(name, ec)
	if err != nil {
		return errors.Wrap(err, "failed to get InstallConfig")
	}

	// populates GetSession()
	// TODO: This needs to become part of ctx object or a secret
	authLocation := filepath.Join(ec.Directory, ".azure", "osServicePrincipal.json")
	err = saveCredentials(icazure.Credentials{
		SubscriptionID: ec.SubscriptionID,
		ClientID:       ec.ClientID,
		ClientSecret:   ec.ClientSecret,
		TenantID:       ec.TenantID,
	}, authLocation)
	if err != nil {
		return err
	}
	if os.Setenv("AZURE_AUTH_LOCATION", authLocation) != nil {
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
	if err := asset.PersistToFile(ic, ec.Directory); err != nil {
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

		if err2 := asset.PersistToFile(a, ec.Directory); err2 != nil {
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
