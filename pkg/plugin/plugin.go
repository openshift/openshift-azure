// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/healthcheck"
	"github.com/openshift/openshift-azure/pkg/initialize"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/upgrade"
	"github.com/openshift/openshift-azure/pkg/validate"
)

type plugin struct {
	entry     *logrus.Entry
	syncImage string
}

var _ api.Plugin = &plugin{}

func NewPlugin(entry *logrus.Entry, syncImage string) api.Plugin {
	log.New(entry)
	return &plugin{
		entry:     entry,
		syncImage: syncImage,
	}
}

func (p *plugin) MergeConfig(ctx context.Context, cs, oldCs *api.OpenShiftManagedCluster) *api.OpenShiftManagedCluster {
	if oldCs == nil {
		return cs
	}
	log.Info("merging internal data models")
	return api.Merge(oldCs, cs)
}

func (p *plugin) Validate(ctx context.Context, new, old *api.OpenShiftManagedCluster, externalOnly bool) []error {
	log.Info("validating internal data models")
	return validate.Validate(new, old, externalOnly)
}

func (p *plugin) GenerateConfig(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("generating configs")
	// TODO should we save off the original config here and if there are any errors we can restore it?
	if cs.Config == nil {
		cs.Config = &api.Config{}
	}

	upgrader := config.NewSimpleUpgrader(p.entry)
	err := upgrader.Upgrade(ctx, cs)
	if err != nil {
		return err
	}

	err = config.Generate(cs)
	if err != nil {
		return err
	}
	if p.syncImage != "" {
		cs.Config.Images.Sync = p.syncImage
	}
	return nil
}

func (p *plugin) GenerateARM(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) ([]byte, error) {
	log.Info("generating arm templates")
	generator := arm.NewSimpleGenerator(p.entry)
	return generator.Generate(ctx, cs, isUpdate)
}

func (p *plugin) InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("initializing cluster")
	initializer := initialize.NewSimpleInitializer(p.entry)
	return initializer.InitializeCluster(ctx, cs)
}

func (p *plugin) HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("starting health check")
	healthChecker := healthcheck.NewSimpleHealthChecker(p.entry)
	return healthChecker.HealthCheck(ctx, cs)
}

func (p *plugin) Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte) error {
	log.Info("starting update")
	upgrader := upgrade.NewSimpleUpgrader(p.entry)
	return upgrader.Update(ctx, cs, azuredeploy)
}
