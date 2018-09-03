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

func (p *plugin) MergeConfig(ctx context.Context, cs, oldCs *acsapi.OpenShiftManagedCluster) {
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
	if len(cs.Properties.RouterProfiles) == 0 {
		cs.Properties.RouterProfiles = oldCs.Properties.RouterProfiles
	}
	if cs.Properties.ServicePrincipalProfile == nil {
		cs.Properties.ServicePrincipalProfile = oldCs.Properties.ServicePrincipalProfile
	}
	if len(cs.Properties.FQDN) == 0 {
		cs.Properties.FQDN = oldCs.Properties.FQDN
	}
}

func (p *plugin) Validate(ctx context.Context, new, old *acsapi.OpenShiftManagedCluster, externalOnly bool) []error {
	log.Info("validating internal data models")
	return validate.Validate(new, old, externalOnly)
}

func (p *plugin) GenerateConfig(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
	log.Info("generating configs")
	// TODO should we save off the original config here and if there are any errors we can restore it?
	if cs.Config == nil {
		cs.Config = &acsapi.Config{}
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
	return nil
}

func (p *plugin) GenerateARM(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) ([]byte, error) {
	log.Info("generating arm templates")
	generator := arm.NewSimpleGenerator(p.entry)
	return generator.Generate(ctx, cs)
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
