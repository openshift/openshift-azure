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
	apiValidator    *api.Validator
}

var _ api.Plugin = &plugin{}

// NewPlugin creates a new plugin instance
func NewPlugin(log *logrus.Entry, pluginConfig *api.PluginConfig) (api.Plugin, []error) {
	p := &plugin{
		log:             log,
		config:          *pluginConfig,
		clusterUpgrader: cluster.NewSimpleUpgrader(log, pluginConfig),
		armGenerator:    arm.NewSimpleGenerator(pluginConfig),
		configGenerator: config.NewSimpleGenerator(pluginConfig),
		apiValidator:    api.NewValidator(pluginConfig.TestConfig.RunningUnderTest),
	}
	// validate plugin config
	errs := p.validateConfig()
	if len(errs) > 0 {
		return nil, errs
	}
	return p, nil
}

func (p *plugin) Validate(ctx context.Context, new, old *api.OpenShiftManagedCluster, externalOnly bool) []error {
	p.log.Info("validating internal data models")
	return p.apiValidator.Validate(new, old, externalOnly)
}

func (p *plugin) GenerateConfig(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	p.log.Info("generating configs")
	// TODO should we save off the original config here and if there are any errors we can restore it?
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
