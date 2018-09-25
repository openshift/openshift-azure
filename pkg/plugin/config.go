package plugin

// PluginConfig contains dynamic plugin configuration
type Config struct {
	// The sync image to use
	SyncImage string `json:"syncImage,omitempty"`

	// The VM image to use
	VmImage string `json:"vmImage,omitempty"`
}
