package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"fmt"
	"context"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/openshift/installer/pkg/asset"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"github.com/openshift/installer/pkg/asset/store"
	targetassets "github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/terraform/exec/plugins"
	"github.com/openshift/installer/pkg/types/defaults"
	openstackvalidation "github.com/openshift/installer/pkg/types/openstack/validation"
	"github.com/openshift/installer/pkg/types/validation"
	destroybootstrap "github.com/openshift/installer/pkg/destroy/bootstrap"

	fakerpconfig "github.com/openshift/openshift-azure/pkg/fakerp/config"
	"github.com/openshift/openshift-azure/pkg/util/installer"
)

var (
	gitCommit = "unknown"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetReportCaller(true)
	log := logrus.NewEntry(logrus.StandardLogger())
	// TODO: This should move to installer /pkg
	if len(os.Args) > 0 {
		base := filepath.Base(os.Args[0])
		cname := strings.TrimSuffix(base, filepath.Ext(base))
		if pluginRunner, ok := plugins.KnownPlugins[cname]; ok {
			pluginRunner()
			return
		}
	}

	if err := run(log); err != nil {
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

func run(log *logrus.Entry) error {
	rootCmd := &cobra.Command{
		Use:  "./azure [component]",
		Long: "Azure Red Hat OpenShift dispatcher",
	}
	rootCmd.PersistentFlags().StringP("action", "a", "Create", "Valid values are [Create, Delete]")
	rootCmd.PersistentFlags().StringP("name", "n", "demo-cluster", "Cluster name value")
	rootCmd.Printf("gitCommit %s\n", gitCommit)
	rootCmd.Execute()

	name, err := rootCmd.Flags().GetString("name")
	if err != nil {
		return err
	}

	//action, err := rootCmd.Flags().GetString("action")
	//if err != nil {
	//	return err
	//}

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
	fmt.Println(err)
	fmt.Println(cfg)

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
		return errors.Wrap(err, "failed to persist osServicePrincipal.json")
	}
	err = os.Setenv("AZURE_AUTH_LOCATION", authLocation) 
	if err != nil {
		return errors.Wrap(err, "failed to set AZURE_AUTH_LOCATION")
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


	// waiting routine
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(ec.Directory, "auth", "kubeconfig"))
				if err != nil {
					logrus.Fatal(errors.Wrap(err, "loading kubeconfig"))
				}

	// TODO: Implement context
	ctx:= context.Background()
	// wait for the cluster to come up
	// TODO: All these should become part of installer code base
	err = installer.WaitForBootstrapComplete(ctx, config, ec.Directory)
	if err!=nil{
		return err
	}
	err = destroybootstrap.Destroy(ec.Directory)
	if err!=nil{
		return err
	}
	err = installer.WaitForInstallComplete(ctx, config, ec.Directory)
	if err!=nil{
		return err
	}
	return nil
}
