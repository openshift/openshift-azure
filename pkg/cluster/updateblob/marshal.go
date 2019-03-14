package updateblob

import (
	"encoding/json"
	"sort"
)

var _ json.Marshaler = &HostnameHashes{}
var _ json.Unmarshaler = &HostnameHashes{}

func (hhm HostnameHashes) MarshalJSON() ([]byte, error) {
	hostnames := make([]string, 0, len(hhm))
	for hostname := range hhm {
		hostnames = append(hostnames, hostname)
	}
	sort.Slice(hostnames, func(i, j int) bool { return hostnames[i] < hostnames[j] })

	slice := make([]hostnameHashes, 0, len(hhm))
	for _, hostname := range hostnames {
		slice = append(slice, hostnameHashes{
			Hostname: hostname,
			Hash:     hhm[hostname],
		})
	}

	return json.Marshal(slice)
}

func (hhm *HostnameHashes) UnmarshalJSON(data []byte) error {
	var slice []hostnameHashes
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*hhm = HostnameHashes{}
	for _, vi := range slice {
		(*hhm)[vi.Hostname] = vi.Hash
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
