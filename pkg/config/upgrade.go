package config

import (
	acsapi "github.com/Azure/acs-engine/pkg/api"
)

const (
	versionLatest = 1
)

func Upgrade(cs *acsapi.ContainerService, c *Config) error {
	return nil
}
