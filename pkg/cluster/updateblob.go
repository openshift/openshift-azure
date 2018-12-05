package cluster

import (
	"bytes"
	"encoding/json"
	"sort"
)

type updateblob map[instanceName]hash

var _ json.Marshaler = &updateblob{}
var _ json.Unmarshaler = &updateblob{}

type vmInfo struct {
	InstanceName instanceName `json:"instanceName,omitempty"`
	ScalesetHash hash         `json:"scalesetHash,omitempty"`
}

func (blob updateblob) MarshalJSON() ([]byte, error) {
	instancenames := make([]instanceName, 0, len(blob))
	for instancename := range blob {
		instancenames = append(instancenames, instancename)
	}
	sort.Slice(instancenames, func(i, j int) bool { return instancenames[i] < instancenames[j] })

	slice := make([]vmInfo, 0, len(blob))
	for _, instancename := range instancenames {
		slice = append(slice, vmInfo{
			InstanceName: instancename,
			ScalesetHash: blob[instancename],
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

	updateBlob := u.updateContainer.GetBlobReference(updateBlobName)
	return updateBlob.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
}

func (u *simpleUpgrader) readUpdateBlob() (updateblob, error) {
	updateBlob := u.updateContainer.GetBlobReference(updateBlobName)
	rc, err := updateBlob.Get(nil)
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

func (u *simpleUpgrader) deleteUpdateBlob() error {
	bsc := u.storageClient.GetBlobService()
	c := bsc.GetContainerReference(updateContainerName)
	bc := c.GetBlobReference(updateBlobName)
	return bc.Delete(nil)
}
