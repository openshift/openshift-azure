// Package api defines the external API for the plugin.
package api

import (
	"context"
)

type Plugin interface {
	// MergeConfig merges new and old config so that no unnecessary config
	// is going to get regenerated during generation. It also handles merging
	// partial user requests to allow reusing the same validation code during
	// upgrades. This method should be the first one called by the RP, before
	// validation and generation.
	MergeConfig(new, old *OpenShiftManagedCluster)

	// Validate exists (a) to be able to place validation logic in a
	// single place in the event of multiple external API versions, and (b) to
	// be able to compare a new API manifest against a pre-existing API manifest
	// (for update, upgrade, etc.)
	// externalOnly indicates that fields set by the RP (FQDN and routerProfile.FQDN)
	// should be excluded.
	Validate(new, old *OpenShiftManagedCluster, externalOnly bool) []error

	// GenerateConfig ensures all the necessary in-cluster config is generated
	// for an Openshift cluster.
	GenerateConfig(cs *OpenShiftManagedCluster) error

	GenerateARM(cs *OpenShiftManagedCluster) ([]byte, error)

	InitializeCluster(ctx context.Context, cs *OpenShiftManagedCluster) error

	HealthCheck(ctx context.Context, cs *OpenShiftManagedCluster) error
}

type Upgrade interface {
	IsReady(nodeName string) (bool, error)
}
