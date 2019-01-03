package cluster

import (
	"bytes"
	"encoding/json"
	"sort"
)

type instanceHashMap map[string][]byte

var _ json.Marshaler = &instanceHashMap{}
var _ json.Unmarshaler = &instanceHashMap{}

type instanceHash struct {
	InstanceName string `json:"instanceName,omitempty"`
	Hash         []byte `json:"hash,omitempty"`
}

func (ihm instanceHashMap) MarshalJSON() ([]byte, error) {
	instancenames := make([]string, 0, len(ihm))
	for instancename := range ihm {
		instancenames = append(instancenames, instancename)
	}
	sort.Slice(instancenames, func(i, j int) bool { return instancenames[i] < instancenames[j] })

	slice := make([]instanceHash, 0, len(ihm))
	for _, instancename := range instancenames {
		slice = append(slice, instanceHash{
			InstanceName: instancename,
			Hash:         ihm[instancename],
		})
	}

	return json.Marshal(slice)
}

func (ihm *instanceHashMap) UnmarshalJSON(data []byte) error {
	var slice []instanceHash
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*ihm = instanceHashMap{}
	for _, vi := range slice {
		(*ihm)[vi.InstanceName] = vi.Hash
	}

	return nil
}

type scalesetHashMap map[string][]byte

var _ json.Marshaler = &scalesetHashMap{}
var _ json.Unmarshaler = &scalesetHashMap{}

type scalesetHash struct {
	ScalesetName string `json:"scalesetName,omitempty"`
	Hash         []byte `json:"hash,omitempty"`
}

func (shm scalesetHashMap) MarshalJSON() ([]byte, error) {
	scalesetnames := make([]string, 0, len(shm))
	for scalesetname := range shm {
		scalesetnames = append(scalesetnames, scalesetname)
	}
	sort.Slice(scalesetnames, func(i, j int) bool { return scalesetnames[i] < scalesetnames[j] })

	slice := make([]scalesetHash, 0, len(shm))
	for _, scalesetname := range scalesetnames {
		slice = append(slice, scalesetHash{
			ScalesetName: scalesetname,
			Hash:         shm[scalesetname],
		})
	}

	return json.Marshal(slice)
}

func (shm *scalesetHashMap) UnmarshalJSON(data []byte) error {
	var slice []scalesetHash
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*shm = scalesetHashMap{}
	for _, vi := range slice {
		(*shm)[vi.ScalesetName] = vi.Hash
	}

	return nil
}

type updateblob struct {
	ScalesetHashes scalesetHashMap `json:"scalesetHashes,omitempty"`
	InstanceHashes instanceHashMap `json:"instanceHashes,omitempty"`
}

func newUpdateBlob() *updateblob {
	return &updateblob{
		ScalesetHashes: scalesetHashMap{},
		InstanceHashes: instanceHashMap{},
	}
}

func (u *simpleUpgrader) writeUpdateBlob(blob *updateblob) error {
	data, err := json.Marshal(blob)
	if err != nil {
		return err
	}

	blobRef := u.updateContainer.GetBlobReference(updateBlobName)
	return blobRef.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
}

func (u *simpleUpgrader) readUpdateBlob() (*updateblob, error) {
	blobRef := u.updateContainer.GetBlobReference(updateBlobName)
	rc, err := blobRef.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	d := json.NewDecoder(rc)

	var b updateblob
	if err := d.Decode(&b); err != nil {
		return nil, err
	}
	if b.ScalesetHashes == nil {
		b.ScalesetHashes = scalesetHashMap{}
	}
	if b.InstanceHashes == nil {
		b.InstanceHashes = instanceHashMap{}
	}

	return &b, nil
}

func (u *simpleUpgrader) deleteUpdateBlob() error {
	bsc := u.storageClient.GetBlobService()
	c := bsc.GetContainerReference(updateContainerName)
	bc := c.GetBlobReference(updateBlobName)
	return bc.Delete(nil)
}
