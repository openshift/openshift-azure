package config

import (
	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

// Generator is an interface for sharing the cluster and plugin configs
type Generator interface {
	Generate(cs *api.OpenShiftManagedCluster, template *pluginapi.Config) error
}

type simpleGenerator struct {
	pluginConfig api.PluginConfig
}

var _ Generator = &simpleGenerator{}

// NewSimpleGenerator creates a struct to hold both the cluster and plugin configs
func NewSimpleGenerator(pluginConfig *api.PluginConfig) Generator {
	return &simpleGenerator{
		pluginConfig: *pluginConfig,
	}
}
