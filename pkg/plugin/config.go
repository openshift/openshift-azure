package plugin

// PluginConfig contains dynamic plugin configuration
type Config struct {
	// The sync image to use
	SyncImage string `json:"syncImage,omitempty"`

	// The node image to use
	NodeImage string `json:"nodeImage,omitempty"`
}
