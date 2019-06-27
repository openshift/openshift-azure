// Package api defines the external API for the plugin.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	targetassets "github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/destroy"
	_ "github.com/openshift/installer/pkg/destroy/azure"
	destroybootstrap "github.com/openshift/installer/pkg/destroy/bootstrap"
	"github.com/openshift/installer/pkg/ipnet"
	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"
	"github.com/openshift/installer/pkg/types/defaults"
	openstackvalidation "github.com/openshift/installer/pkg/types/openstack/validation"
	"github.com/openshift/installer/pkg/types/validation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"

	aroassets "github.com/openshift/openshift-azure/pkg/assets"
	"github.com/openshift/openshift-azure/pkg/util/installer"
)

type contextKey string

const (
	ContextClientID       contextKey = "ClientID"
	ContextClientSecret   contextKey = "ClientSecret"
	ContextTenantID       contextKey = "TenantID"
	ContextSubscriptionID contextKey = "SubscriptionID"
)

// Plugin is the main interface to openshift-azure
type Plugin interface {
	// GenerateConfig ensures all the necessary in-cluster config is generated
	// for an Openshift cluster.
	GenerateConfig(ctx context.Context, name string) (*types.InstallConfig, error)

	// ValidateConfig validates the given install config
	ValidateConfig(ctx context.Context, cfg *types.InstallConfig) error

	// Create deploys the cluster
	Create(ctx context.Context, log *logrus.Entry, name string, cfg *types.InstallConfig) error

	// Delete destroys the cluster
	Delete(ctx context.Context, log *logrus.Entry, name string) error
}

type plugin struct {
	directory string
	store     asset.Store
}

// NewPlugin creates a new plugin instance
func NewPlugin(directory string, store asset.Store) Plugin {
	return &plugin{
		directory: directory,
		store:     store,
	}
}

func (p *plugin) ValidateConfig(ctx context.Context, cfg *types.InstallConfig) error {
	if cfg == nil {
		return fmt.Errorf("InstallConfig required, but was nil")
	}
	if cfg.Platform.Azure == nil {
		return fmt.Errorf("Platform Azure must be specified")
	}

	allErrs := validation.ValidateInstallConfig(cfg, openstackvalidation.NewValidValuesFetcher())
	if cfg.Platform.Azure.Region == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("Platform", "Azure", "Region"), "azure region"))
	}
	if cfg.Platform.Azure.BaseDomainResourceGroupName == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("Platform", "Azure", "BaseDomainResourceGroupName"), "azure resource group where the domain is"))
	}
	return allErrs.ToAggregate()
}

func (p *plugin) GenerateConfig(ctx context.Context, name string) (*types.InstallConfig, error) {
	cfg := types.InstallConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: types.InstallConfigVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Compute: []types.MachinePool{
			{
				Name:           "worker",
				Replicas:       to.Int64Ptr(3),
				Hyperthreading: types.HyperthreadingEnabled,
				Platform: types.MachinePoolPlatform{
					Azure: &azuretypes.MachinePool{
						InstanceType: "Standard_DS4_v2",
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
	}

	defaults.SetInstallConfigDefaults(&cfg)
	return &cfg, nil
}

func (p *plugin) setupServicePrincipal(ctx context.Context) error {
	// populates the required credentials for GetSession()
	authLocation := filepath.Join(p.directory, ".azure", "osServicePrincipal.json")
	jsonCreds, err := json.Marshal(icazure.Credentials{
		SubscriptionID: ctx.Value(ContextSubscriptionID).(string),
		ClientID:       ctx.Value(ContextClientID).(string),
		ClientSecret:   ctx.Value(ContextClientSecret).(string),
		TenantID:       ctx.Value(ContextTenantID).(string),
	})
	err = os.MkdirAll(filepath.Dir(authLocation), 0700)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(authLocation, jsonCreds, 0600)

	if err != nil {
		return errors.Wrap(err, "failed to persist osServicePrincipal.json")
	}
	err = os.Setenv("AZURE_AUTH_LOCATION", authLocation)
	if err != nil {
		return errors.Wrap(err, "failed to set AZURE_AUTH_LOCATION")
	}
	return nil
}

func (p *plugin) Create(ctx context.Context, log *logrus.Entry, name string, cfg *types.InstallConfig) error {
	log.Infof("Creating cluster %s", name)
	err := p.setupServicePrincipal(ctx)
	if err != nil {
		return err
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
	if err := asset.PersistToFile(ic, p.directory); err != nil {
		return errors.Wrap(err, "failed to write install config")
	}

	targets := targetassets.InstallConfig
	targets = append(targets, targetassets.IgnitionConfigs...)
	targets = append(targets, targetassets.Manifests...)
	// inject azure assets
	targets = append(targets, aroassets.AroManifests...)
	targets = append(targets, targetassets.Cluster...)

	for _, a := range targets {
		err := p.store.Fetch(a, targets...)
		if err != nil {
			err = errors.Wrapf(err, "failed to fetch %s", a.Name())
		}

		if err2 := asset.PersistToFile(a, p.directory); err2 != nil {
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
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(p.directory, "auth", "kubeconfig"))
	if err != nil {
		logrus.Fatal(errors.Wrap(err, "loading kubeconfig"))
	}

	// wait for the cluster to come up
	// TODO: All these should become part of installer code base
	err = installer.WaitForBootstrapComplete(ctx, config, p.directory)
	if err != nil {
		return err
	}
	err = destroybootstrap.Destroy(p.directory)
	if err != nil {
		return err
	}
	err = installer.WaitForInstallComplete(ctx, config, p.directory)
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) Delete(ctx context.Context, log *logrus.Entry, name string) error {
	log.Infof("Deleting cluster %s", name)
	authLocation := filepath.Join(p.directory, ".azure", "osServicePrincipal.json")
	err := os.Setenv("AZURE_AUTH_LOCATION", authLocation)
	if err != nil {
		return errors.Wrap(err, "failed to set AZURE_AUTH_LOCATION")
	}
	destroyer, err := destroy.New(log, p.directory)
	if err != nil {
		return errors.Wrap(err, "Failed while preparing to destroy cluster")
	}
	if err := destroyer.Run(); err != nil {
		return errors.Wrap(err, "Failed to destroy cluster")
	}

	for _, asset := range targetassets.Cluster {
		if err := p.store.Destroy(asset); err != nil {
			return errors.Wrapf(err, "failed to destroy asset %q", asset.Name())
		}
	}
	// delete the state file as well
	err = p.store.DestroyState()
	if err != nil {
		return errors.Wrap(err, "failed to remove state file")
	}

	// delete all temporary files
	err = os.RemoveAll(p.directory)
	if err != nil {
		return err
	}

	return nil

}
