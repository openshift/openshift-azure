package api

import (
	"encoding/json"
)

func (in *OpenShiftManagedCluster) DeepCopy() (out *OpenShiftManagedCluster) {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		panic(err)
	}

	return
}

func (in *Config) DeepCopy() (out *Config) {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		panic(err)
	}

	return
}
