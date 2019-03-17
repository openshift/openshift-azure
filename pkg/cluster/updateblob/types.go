package updateblob

type HostnameHashes map[string][]byte

type hostnameHashes struct {
	Hostname string `json:"hostname,omitempty"`
	Hash     []byte `json:"hash,omitempty"`
}

type ScalesetHashes map[string][]byte

type scalesetHashes struct {
	ScalesetName string `json:"scalesetName,omitempty"`
	Hash         []byte `json:"hash,omitempty"`
}

type UpdateBlob struct {
	// ScalesetHashes stores the config hash for each worker scaleset
	ScalesetHashes ScalesetHashes `json:"scalesetHashes,omitempty"`
	// HostnameHashes stores the config hash for each master instance
	HostnameHashes HostnameHashes `json:"hostnameHashes,omitempty"`
}

func NewUpdateBlob() *UpdateBlob {
	return &UpdateBlob{
		ScalesetHashes: ScalesetHashes{},
		HostnameHashes: HostnameHashes{},
	}
}
