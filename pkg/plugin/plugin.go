// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/v1"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/healthcheck"
	"github.com/openshift/openshift-azure/pkg/validate"
)

type Plugin struct{}

var _ api.Plugin = &Plugin{}

func (p *Plugin) ValidateExternal(oc *v1.OpenShiftCluster) []error {
	return validate.OpenShiftCluster(oc)
}

func (p *Plugin) ValidateInternal(new, old *acsapi.ContainerService) []error {
	return validate.ContainerService(new, old)
}

func (p *Plugin) GenerateConfig(cs *acsapi.ContainerService) error {
	// TODO should we save off the original config here and if there are any errors we can restore it?
	if cs.Config == nil {
		cs.Config = &acsapi.Config{}
	}

	err := config.Upgrade(cs)
	if err != nil {
		return err
	}

	err = config.Generate(cs)
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) GenerateARM(cs *acsapi.ContainerService) ([]byte, error) {
	return arm.Generate(cs)
}

func (p *Plugin) HealthCheck(ctx context.Context, cs *acsapi.ContainerService) error {
	return healthcheck.HealthCheck(ctx, cs)
}
