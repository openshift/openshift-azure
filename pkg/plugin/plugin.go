// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/upgrade"
)

type plugin struct {
	entry           *logrus.Entry
	config          api.PluginConfig
	configUpgrader  config.Upgrader
	clusterUpgrader upgrade.Upgrader
	armGenerator    arm.Generator
}

var _ api.Plugin = &plugin{}

// NewPlugin creates a new plugin instance
func NewPlugin(entry *logrus.Entry, pluginConfig api.PluginConfig) api.Plugin {
	log.New(entry)
	return &plugin{
		entry:           entry,
		config:          pluginConfig,
		configUpgrader:  config.NewSimpleUpgrader(entry),
		clusterUpgrader: upgrade.NewSimpleUpgrader(entry, pluginConfig),
		armGenerator:    arm.NewSimpleGenerator(entry),
	}
}

func (p *plugin) MergeConfig(ctx context.Context, cs, oldCs *api.OpenShiftManagedCluster) {
	if oldCs == nil {
		return
	}
	log.Info("merging internal data models")

	// generated config should be copied as is
	old := oldCs.DeepCopy()
	cs.Config = old.Config

	// user request data
	// need to merge partial requests
	if len(cs.Properties.AgentPoolProfiles) == 0 {
		cs.Properties.AgentPoolProfiles = oldCs.Properties.AgentPoolProfiles
	}
	if len(cs.Properties.OpenShiftVersion) == 0 {
		cs.Properties.OpenShiftVersion = oldCs.Properties.OpenShiftVersion
	}
	if len(cs.Properties.PublicHostname) == 0 {
		cs.Properties.PublicHostname = oldCs.Properties.PublicHostname
	}
	if cs.Properties.NetworkProfile == nil {
		cs.Properties.NetworkProfile = oldCs.Properties.NetworkProfile
	}
	if len(cs.Properties.RouterProfiles) == 0 {
		cs.Properties.RouterProfiles = oldCs.Properties.RouterProfiles
	}
	if cs.Properties.ServicePrincipalProfile == nil {
		cs.Properties.ServicePrincipalProfile = oldCs.Properties.ServicePrincipalProfile
	}
	if cs.Properties.AzProfile == nil {
		cs.Properties.AzProfile = oldCs.Properties.AzProfile
	}
	if cs.Properties.AuthProfile == nil {
		cs.Properties.AuthProfile = oldCs.Properties.AuthProfile
	}
	if len(cs.Properties.FQDN) == 0 {
		cs.Properties.FQDN = oldCs.Properties.FQDN
	}
}

func (p *plugin) Validate(ctx context.Context, new, old *api.OpenShiftManagedCluster, externalOnly bool) []error {
	log.Info("validating internal data models")
	return api.Validate(new, old, externalOnly)
}

func (p *plugin) GenerateConfig(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("generating configs")
	// TODO should we save off the original config here and if there are any errors we can restore it?
	if cs.Config == nil {
		cs.Config = &api.Config{}
	}

	err := p.configUpgrader.Upgrade(ctx, cs)
	if err != nil {
		return err
	}

	err = config.Generate(cs, p.config)
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) GenerateARM(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) ([]byte, error) {
	log.Info("generating arm templates")
	return p.armGenerator.Generate(ctx, cs, isUpdate)
}

func (p *plugin) InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("initializing cluster")
	return p.clusterUpgrader.InitializeCluster(ctx, cs)
}

func (p *plugin) HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("starting health check")
	return p.clusterUpgrader.HealthCheck(ctx, cs)
}

func (p *plugin) CreateOrUpdate(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, isUpdate bool, deployFn api.DeployFn) error {
	var err error
	if isUpdate {
		log.Info("starting update")
		err = p.clusterUpgrader.Update(ctx, cs, azuredeploy, deployFn)
	} else {
		log.Info("starting deploy")
		err = p.clusterUpgrader.Deploy(ctx, cs, azuredeploy, deployFn)
	}
	if err != nil {
		return err
	}

	// Wait for infrastructure services to be healthy
	err = p.clusterUpgrader.WaitForInfraServices(ctx, cs)
	if err != nil {
		return err
	}

	return p.HealthCheck(ctx, cs)
}
