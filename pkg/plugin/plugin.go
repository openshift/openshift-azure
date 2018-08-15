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
	"github.com/openshift/openshift-azure/pkg/log"
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

func (p *plugin) ValidateInternal(new, old *acsapi.ContainerService) []error {
	log.Info("validating internal data models")
	return validate.ContainerService(new, old)
}

func (p *plugin) GenerateConfig(cs *acsapi.ContainerService) error {
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

func (p *plugin) GenerateARM(cs *acsapi.ContainerService) ([]byte, error) {
	log.Info("generating arm templates")
	generator := arm.NewSimpleGenerator(p.entry)
	return generator.Generate(cs)
}

func (p *plugin) HealthCheck(ctx context.Context, cs *acsapi.ContainerService) error {
	log.Info("starting health check")
	healthChecker := healthcheck.NewSimpleHealthChecker(p.entry)
	return healthChecker.HealthCheck(ctx, cs)
}
