// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/healthcheck"
	"github.com/openshift/openshift-azure/pkg/initialize"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/upgrade"
)

type plugin struct {
	entry  *logrus.Entry
	config api.PluginConfig
}

var _ api.Plugin = &plugin{}

func getEnv(name string, defaultValue ...string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return value
}

// GetRequiredConfigEnvVars exposes the required variables to build a plugin config
func GetRequiredConfigEnvVars() []string {
	return []string{
		"AZURE_TENANT_ID",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"RESOURCEGROUP",
	}
}

// NewPluginConfigFromEnv loads all config items from os env vars
func NewPluginConfigFromEnv() (api.PluginConfig, error) {
	pc := api.PluginConfig{}
	required := GetRequiredConfigEnvVars()

	var missing []string
	for i := range required {
		value := getEnv(required[i])
		if len(value) == 0 {
			missing = append(missing, required[i])
		}
	}

	if len(missing) > 0 {
		return pc, fmt.Errorf("can't build plugin config from env, missing: %s", missing)
	}

	// fill out the config struct
	acceptAgreement := false
	if getEnv("AUTOACCEPT_MARKETPLACE_AGREEMENT", "yes") == "yes" {
		acceptAgreement = true
	}
	pc.AcceptLanguages = strings.Split(getEnv("ACCEPT_LANGUAGES", "en-us"), ",")
	pc.SyncImage = getEnv("SYNC_IMAGE", "sync:latest")
	pc.AzTenantID = getEnv("AZURE_TENANT_ID")
	pc.AzSubscriptionID = getEnv("AZURE_SUBSCRIPTION_ID")
	pc.AzClientID = getEnv("AZURE_CLIENT_ID")
	pc.AzClientSecret = getEnv("AZURE_CLIENT_SECRET")
	pc.ResourceGroup = getEnv("RESOURCEGROUP")
	pc.AcceptMarketplaceAgreement = acceptAgreement
	pc.DNSDomain = getEnv("DNS_DOMAIN", "osadev.cloud")

	return pc, nil
}

// NewPlugin creates a new plugin instance
func NewPlugin(entry *logrus.Entry, pluginConfig api.PluginConfig) api.Plugin {
	log.New(entry)
	return &plugin{
		entry:  entry,
		config: pluginConfig,
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

	upgrader := config.NewSimpleUpgrader(p.entry)
	err := upgrader.Upgrade(ctx, cs)
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
	generator := arm.NewSimpleGenerator(p.entry)
	return generator.Generate(ctx, cs, isUpdate)
}

func (p *plugin) InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("initializing cluster")
	initializer := initialize.NewSimpleInitializer(p.entry, p.config)
	return initializer.InitializeCluster(ctx, cs)
}

func (p *plugin) HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("starting health check")
	healthChecker := healthcheck.NewSimpleHealthChecker(p.entry, p.config)
	return healthChecker.HealthCheck(ctx, cs)
}

func (p *plugin) Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte) error {
	log.Info("starting update")
	upgrader := upgrade.NewSimpleUpgrader(p.entry, p.config)
	return upgrader.Update(ctx, cs, azuredeploy, p.config)
}
