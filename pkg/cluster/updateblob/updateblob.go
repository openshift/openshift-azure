package updateblob

import (
	"encoding/json"
	"sort"
)

type InstanceName string
type Hash string

type Updateblob map[InstanceName]Hash

var _ json.Marshaler = &Updateblob{}
var _ json.Unmarshaler = &Updateblob{}

type vmInfo struct {
	InstanceName InstanceName `json:"instanceName,omitempty"`
	ScalesetHash Hash         `json:"scalesetHash,omitempty"`
}

func (blob Updateblob) MarshalJSON() ([]byte, error) {
	instancenames := make([]InstanceName, 0, len(blob))
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

func (blob *Updateblob) UnmarshalJSON(data []byte) error {
	var slice []vmInfo
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*blob = Updateblob{}
	for _, vi := range slice {
		(*blob)[vi.InstanceName] = vi.ScalesetHash
	}

	return nil
}
