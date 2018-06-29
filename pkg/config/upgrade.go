package config

import (
	"github.com/ghodss/yaml"

	"github.com/jim-minter/azure-helm/pkg/api"
)

const (
	versionLatest = 1
)

func Upgrade(m *api.Manifest, c *Config) ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return b, nil
}
