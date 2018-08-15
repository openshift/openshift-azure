// Package api defines the external API for the plugin.
package api

import (
	"context"
)

type Plugin interface {
	// Validate exists (a) to be able to place validation logic in a
	// single place in the event of multiple external API versions, and (b) to
	// be able to compare a new API manifest against a pre-existing API manifest
	// (for update, upgrade, etc.)
	// externalOnly indicates that fields set by the RP (FQDN and routerProfile.FQDN)
	// should be excluded.
	Validate(new, old *OpenShiftManagedCluster, externalOnly bool) []error

	GenerateConfig(cs *OpenShiftManagedCluster) error

	GenerateARM(cs *OpenShiftManagedCluster) ([]byte, error)

	HealthCheck(ctx context.Context, cs *OpenShiftManagedCluster) error
}

type Upgrade interface {
	IsReady(nodeName string) (bool, error)
}
