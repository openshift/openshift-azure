// Package api defines the external API for the plugin.
package api

import (
	"context"
)

type contextKey string

const (
	ContextKeyClientAuthorizer      contextKey = "ClientAuthorizer"
	ContextKeyVaultClientAuthorizer contextKey = "VaultClientAuthorizer"
	ContextAcceptLanguages          contextKey = "AcceptLanguages"
)

// Plugin is the main interface to openshift-azure
type Plugin interface {
	// GenerateConfig ensures all the necessary in-cluster config is generated
	// for an Openshift cluster.
	GenerateConfig(ctx context.Context) error

	// CreateOrUpdate either deploys or runs the update depending on the isUpdate argument
	// this will call the deployer.
	Create(ctx context.Context) *error
}
