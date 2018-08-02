// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"

	"github.com/ghodss/yaml"

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

func (p *Plugin) GenerateConfig(cs *acsapi.ContainerService, configBytes []byte) ([]byte, error) {
	var c *config.Config
	if len(configBytes) > 0 {
		err := yaml.Unmarshal(configBytes, &c)
		if err != nil {
			return nil, err
		}
	} else {
		c = &config.Config{}
	}

	err := config.Upgrade(cs, c)
	if err != nil {
		return nil, err
	}

	err = config.Generate(cs, c)
	if err != nil {
		return nil, err
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return b, err
}

func (p *Plugin) GenerateARM(cs *acsapi.ContainerService, configBytes []byte) ([]byte, error) {
	var c *config.Config
	err := yaml.Unmarshal(configBytes, &c)
	if err != nil {
		return nil, err
	}

	return arm.Generate(cs, c)
}

func (p *Plugin) HealthCheck(ctx context.Context, cs *acsapi.ContainerService, configBytes []byte) error {
	var c *config.Config
	err := yaml.Unmarshal(configBytes, &c)
	if err != nil {
		return err
	}

	return healthcheck.HealthCheck(ctx, cs, c)
}
