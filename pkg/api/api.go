// Package api defines the external API for the plugin.
package api

type Manifest struct {
	TenantID               string
	SubscriptionID         string
	ClientID               string
	ClientSecret           string
	Location               string
	ResourceGroup          string
	VMSize                 string
	ComputeCount           int
	InfraCount             int
	RoutingConfigSubdomain string
	PublicHostname         string
}

type NewPlugin func(manifestBytes, oldManifestBytes, configBytes []byte) (Plugin, error)

type Plugin interface {
	Validate() error
	GenerateConfig() ([]byte, error)
	GenerateHelm() ([]byte, error)
	GenerateARM() ([]byte, error)
	HealthCheck() error
}
