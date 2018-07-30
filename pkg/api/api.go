// Package api defines the external API for the plugin.
package api

import (
	"context"

	acsapi "github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/osa/vlabs"
)

type Plugin interface {
	ValidateExternal(oc *vlabs.OpenShiftCluster) []error

	// ValidateInternal exists (a) to be able to place validation logic in a
	// single place in the event of multiple external API versions, and (b) to
	// be able to compare a new API manifest against a pre-existing API manifest
	// (for update, upgrade, etc.)

	// TODO: confirm with MSFT that they can pass in `old` at the time
	// ValidateInternal is called and that it makes sense to do this.
	ValidateInternal(new, old *acsapi.ContainerService) []error

	GenerateConfig(cs *acsapi.ContainerService, configBytes []byte) ([]byte, error)

	GenerateARM(cs *acsapi.ContainerService, configBytes []byte) ([]byte, error)

	HealthCheck(ctx context.Context, cs *acsapi.ContainerService, configBytes []byte) error
}
