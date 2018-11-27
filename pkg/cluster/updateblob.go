package cluster

import (
	"bytes"
	"encoding/json"
)

type updateblob map[instanceName]hash

var _ json.Marshaler = &updateblob{}
var _ json.Unmarshaler = &updateblob{}

type vmInfo struct {
	InstanceName instanceName `json:"instanceName,omitempty"`
	ScalesetHash hash         `json:"scalesetHash,omitempty"`
}

func (blob updateblob) MarshalJSON() ([]byte, error) {
	slice := make([]vmInfo, 0, len(blob))
	for instancename, hash := range blob {
		slice = append(slice, vmInfo{
			InstanceName: instancename,
			ScalesetHash: hash,
		})
	}

	return json.Marshal(slice)
}

func (blob *updateblob) UnmarshalJSON(data []byte) error {
	var slice []vmInfo
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*blob = updateblob{}
	for _, vi := range slice {
		(*blob)[vi.InstanceName] = vi.ScalesetHash
	}

	return nil
}

func (u *simpleUpgrader) writeUpdateBlob(blob updateblob) error {
	data, err := json.Marshal(blob)
	if err != nil {
		return err
	}
	return u.updateBlob.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
}

func (u *simpleUpgrader) readUpdateBlob() (updateblob, error) {
	rc, err := u.updateBlob.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	d := json.NewDecoder(rc)

	b := updateblob{}
	if err := d.Decode(&b); err != nil {
		return nil, err
	}

	return b, nil
}
