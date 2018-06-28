package config

import "github.com/jim-minter/azure-helm/pkg/api"

const (
	versionLatest = 1
)

func Upgrade(m *api.Manifest, c *Config) (*Config, error) {
	return c, nil
}
