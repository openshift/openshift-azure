package updateblob

import (
	"encoding/json"
	"sort"
)

var _ json.Marshaler = &InstanceHashes{}
var _ json.Unmarshaler = &InstanceHashes{}

func (ihm InstanceHashes) MarshalJSON() ([]byte, error) {
	instancenames := make([]string, 0, len(ihm))
	for instancename := range ihm {
		instancenames = append(instancenames, instancename)
	}
	sort.Slice(instancenames, func(i, j int) bool { return instancenames[i] < instancenames[j] })

	slice := make([]instanceHashes, 0, len(ihm))
	for _, instancename := range instancenames {
		slice = append(slice, instanceHashes{
			InstanceName: instancename,
			Hash:         ihm[instancename],
		})
	}

	return json.Marshal(slice)
}

func (ihm *InstanceHashes) UnmarshalJSON(data []byte) error {
	var slice []instanceHashes
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*ihm = InstanceHashes{}
	for _, vi := range slice {
		(*ihm)[vi.InstanceName] = vi.Hash
	}

	return nil
}

var _ json.Marshaler = &ScalesetHashes{}
var _ json.Unmarshaler = &ScalesetHashes{}

func (shm ScalesetHashes) MarshalJSON() ([]byte, error) {
	scalesetnames := make([]string, 0, len(shm))
	for scalesetname := range shm {
		scalesetnames = append(scalesetnames, scalesetname)
	}
	sort.Slice(scalesetnames, func(i, j int) bool { return scalesetnames[i] < scalesetnames[j] })

	slice := make([]scalesetHashes, 0, len(shm))
	for _, scalesetname := range scalesetnames {
		slice = append(slice, scalesetHashes{
			ScalesetName: scalesetname,
			Hash:         shm[scalesetname],
		})
	}

	return json.Marshal(slice)
}

func (shm *ScalesetHashes) UnmarshalJSON(data []byte) error {
	var slice []scalesetHashes
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*shm = ScalesetHashes{}
	for _, vi := range slice {
		(*shm)[vi.ScalesetName] = vi.Hash
	}

	return nil
}
