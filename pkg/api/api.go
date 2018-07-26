// Package api defines the external API for the plugin.
package api

import (
	"context"

	acsapi "github.com/Azure/acs-engine/pkg/api"
)

type NewPlugin func(cs, oldCs *acsapi.ContainerService, configBytes []byte) (Plugin, error)

type Plugin interface {
	Validate() error
	GenerateConfig() ([]byte, error)
	GenerateARM() ([]byte, error)
	HealthCheck(context.Context) error
}
