// Package api defines the external API for the plugin.
package api

import (
	"context"
)

type Plugin interface {
	// ValidateInternal exists (a) to be able to place validation logic in a
	// single place in the event of multiple external API versions, and (b) to
	// be able to compare a new API manifest against a pre-existing API manifest
	// (for update, upgrade, etc.)

	// TODO: confirm with MSFT that they can pass in `old` at the time
	// ValidateInternal is called and that it makes sense to do this.
	ValidateInternal(new, old *ContainerService) []error

	GenerateConfig(cs *ContainerService) error

	GenerateARM(cs *ContainerService) ([]byte, error)

	HealthCheck(ctx context.Context, cs *ContainerService) error
}
