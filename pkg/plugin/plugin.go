// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/healthcheck"
	"github.com/openshift/openshift-azure/pkg/initialize"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/upgrade"
	"github.com/openshift/openshift-azure/pkg/validate"
)

type plugin struct {
	entry *logrus.Entry
}

var _ api.Plugin = &plugin{}

func NewPlugin(entry *logrus.Entry) api.Plugin {
	log.New(entry)
	return &plugin{
		entry: entry,
	}
}

func (p *plugin) MergeConfig(cs, oldCs *acsapi.OpenShiftManagedCluster) {
	if oldCs == nil {
		return
	}
	log.Info("merging internal data models")

	// generated config should be copied as is
	cs.Config = oldCs.Config

	// user request data
	// need to merge partial requests
	if len(cs.Properties.AgentPoolProfiles) == 0 {
		cs.Properties.AgentPoolProfiles = oldCs.Properties.AgentPoolProfiles
	}
	if cs.Properties.OrchestratorProfile == nil {
		cs.Properties.OrchestratorProfile = oldCs.Properties.OrchestratorProfile
	}
	if len(cs.Properties.OrchestratorProfile.OrchestratorVersion) == 0 {
		cs.Properties.OrchestratorProfile.OrchestratorVersion = oldCs.Properties.OrchestratorProfile.OrchestratorVersion
	}
	if cs.Properties.OrchestratorProfile.OpenShiftConfig == nil {
		cs.Properties.OrchestratorProfile.OpenShiftConfig = oldCs.Properties.OrchestratorProfile.OpenShiftConfig
	}
	if len(cs.Properties.FQDN) == 0 {
		cs.Properties.FQDN = oldCs.Properties.FQDN
	}
}

func (p *plugin) Validate(new, old *acsapi.OpenShiftManagedCluster, externalOnly bool) []error {
	log.Info("validating internal data models")
	return validate.Validate(new, old, externalOnly)
}

func (p *plugin) GenerateConfig(cs *acsapi.OpenShiftManagedCluster) error {
	log.Info("generating configs")
	// TODO should we save off the original config here and if there are any errors we can restore it?
	if cs.Config == nil {
		cs.Config = &acsapi.Config{}
	}

	upgrader := config.NewSimpleUpgrader(p.entry)
	err := upgrader.Upgrade(cs)
	if err != nil {
		return err
	}

	err = config.Generate(cs)
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) GenerateARM(cs *acsapi.OpenShiftManagedCluster) ([]byte, error) {
	log.Info("generating arm templates")
	generator := arm.NewSimpleGenerator(p.entry)
	return generator.Generate(cs)
}

func (p *plugin) InitializeCluster(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
	log.Info("initializing cluster")
	initializer := initialize.NewSimpleInitializer(p.entry)
	return initializer.InitializeCluster(ctx, cs)
}

func (p *plugin) HealthCheck(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
	log.Info("starting health check")
	healthChecker := healthcheck.NewSimpleHealthChecker(p.entry)
	return healthChecker.HealthCheck(ctx, cs)
}

func (p *plugin) Drain(ctx context.Context, cs *acsapi.OpenShiftManagedCluster, role api.AgentPoolProfileRole, nodeName string) error {
	upgrader := upgrade.NewSimpleUpgrader(p.entry)
	return upgrader.Drain(ctx, cs, role, nodeName)
}

func (p *plugin) WaitForReady(ctx context.Context, cs *acsapi.OpenShiftManagedCluster, role api.AgentPoolProfileRole, nodeName string) error {
	upgrader := upgrade.NewSimpleUpgrader(p.entry)
	return upgrader.WaitForReady(ctx, cs, role, nodeName)
}
