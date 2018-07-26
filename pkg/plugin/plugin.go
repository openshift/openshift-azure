// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	acsapi "github.com/Azure/acs-engine/pkg/api"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/healthcheck"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/validate"
)

type Plugin struct {
	cs     *acsapi.ContainerService
	oldCs  *acsapi.ContainerService
	config *config.Config
}

var _ api.Plugin = &Plugin{}

func NewPlugin(cs, oldCs *acsapi.ContainerService, configBytes []byte) (api.Plugin, error) {
	var config *config.Config
	err := yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		cs:     cs,
		oldCs:  oldCs,
		config: config,
	}, nil
}

func (p *Plugin) Validate() error {
	return validate.Validate(p.cs, p.oldCs)
}

func (p *Plugin) GenerateConfig() ([]byte, error) {
	if p.config == nil {
		p.config = &config.Config{}
	}

	err := config.Upgrade(p.cs, p.config)
	if err != nil {
		return nil, err
	}

	err = config.Generate(p.cs, p.config)
	if err != nil {
		return nil, err
	}

	b, err := yaml.Marshal(p.config)
	if err != nil {
		return nil, err
	}
	return b, err
}

func (p *Plugin) GenerateARM() ([]byte, error) {
	return arm.Generate(p.cs, p.config)
}

func (p *Plugin) HealthCheck(ctx context.Context) error {
	return healthcheck.HealthCheck(ctx, p.cs, p.config)
}

// WriteHelpers is for development - not part of the external API
func (p *Plugin) WriteHelpers() error {
	b, err := tls.PrivateKeyAsBytes(p.config.SSHKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/id_rsa", b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(p.config.AdminKubeconfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("_data/_out/admin.kubeconfig", b, 0600)
}

// Enrich is not part of the external API
func Enrich(cs *acsapi.ContainerService) error {
	cs.Properties.AzProfile = &acsapi.AzProfile{
		TenantID:       os.Getenv("AZURE_TENANT_ID"),
		SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		ResourceGroup:  os.Getenv("RESOURCEGROUP"),
	}

	if cs.Properties.AzProfile.TenantID == "" {
		return fmt.Errorf("must set AZURE_TENANT_ID")
	}
	if cs.Properties.AzProfile.SubscriptionID == "" {
		return fmt.Errorf("must set AZURE_SUBSCRIPTION_ID")
	}
	if cs.Properties.AzProfile.ResourceGroup == "" {
		return fmt.Errorf("must set RESOURCEGROUP")
	}

	return nil
}
