package updateblob

type InstanceHashes map[string][]byte

type instanceHashes struct {
	InstanceName string `json:"instanceName,omitempty"`
	Hash         []byte `json:"hash,omitempty"`
}

type ScalesetHashes map[string][]byte

type scalesetHashes struct {
	ScalesetName string `json:"scalesetName,omitempty"`
	Hash         []byte `json:"hash,omitempty"`
}

type UpdateBlob struct {
	ScalesetHashes ScalesetHashes `json:"scalesetHashes,omitempty"`
	InstanceHashes InstanceHashes `json:"instanceHashes,omitempty"`
}

func NewUpdateBlob() *UpdateBlob {
	return &UpdateBlob{
		ScalesetHashes: ScalesetHashes{},
		InstanceHashes: InstanceHashes{},
	}
}
