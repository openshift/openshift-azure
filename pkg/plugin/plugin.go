// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/config"
)

type plugin struct {
	log             *logrus.Entry
	config          api.PluginConfig
	clusterUpgrader cluster.Upgrader
	armGenerator    arm.Generator
	configGenerator config.Generator
}

var _ api.Plugin = &plugin{}

// NewPlugin creates a new plugin instance
func NewPlugin(log *logrus.Entry, pluginConfig *api.PluginConfig) api.Plugin {
	return &plugin{
		log:             log,
		config:          *pluginConfig,
		clusterUpgrader: cluster.NewSimpleUpgrader(log, pluginConfig),
		armGenerator:    arm.NewSimpleGenerator(pluginConfig),
		configGenerator: config.NewSimpleGenerator(pluginConfig),
	}
}

func (p *plugin) MergeConfig(ctx context.Context, cs, oldCs *api.OpenShiftManagedCluster) {
	if oldCs == nil {
		return
	}
	p.log.Info("merging internal data models")

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
	p.log.Info("validating internal data models")
	return api.Validate(new, old, externalOnly)
}

func (p *plugin) GenerateConfig(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	p.log.Info("generating configs")
	// TODO should we save off the original config here and if there are any errors we can restore it?
	if cs.Config == nil {
		cs.Config = &api.Config{}
	}

	err := p.configGenerator.Generate(cs)
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) GenerateARM(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) (map[string]interface{}, error) {
	p.log.Info("generating arm templates")
	return p.armGenerator.Generate(ctx, cs, isUpdate)
}

func (p *plugin) CreateOrUpdate(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, isUpdate bool, deployFn api.DeployFn) *api.PluginError {
	if isUpdate {
		p.log.Info("starting update")
		if err := p.clusterUpgrader.Update(ctx, cs, azuretemplate, deployFn); err != nil {
			return err
		}
	} else {
		p.log.Info("starting deploy")
		if err := p.clusterUpgrader.Deploy(ctx, cs, azuretemplate, deployFn); err != nil {
			return err
		}
	}

	// Wait for infrastructure services to be healthy
	p.log.Info("waiting for infra services to be ready")
	if err := p.clusterUpgrader.WaitForInfraServices(ctx, cs); err != nil {
		return err
	}

	p.log.Info("starting health check")
	if err := p.clusterUpgrader.HealthCheck(ctx, cs); err != nil {
		return err
	}

	// explicitly return nil if all went well
	return nil
}
