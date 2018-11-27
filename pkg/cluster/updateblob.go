package cluster

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

type updateblob map[instanceName]hash

type vmInfo struct {
	InstanceName instanceName `json:"instanceName,omitempty"`
	ScalesetHash hash         `json:"scalesetHash,omitempty"`
}

func (u *simpleUpgrader) writeUpdateBlob(b updateblob) error {
	blob := make([]vmInfo, 0, len(b))
	for instancename, hash := range b {
		blob = append(blob, vmInfo{
			InstanceName: instancename,
			ScalesetHash: hash,
		})
	}
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

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var blob []vmInfo
	if err := json.Unmarshal(data, &blob); err != nil {
		return nil, err
	}
	b := updateblob{}
	for _, vi := range blob {
		b[vi.InstanceName] = vi.ScalesetHash
	}
	return b, nil
}
