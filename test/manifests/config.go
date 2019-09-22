package manifests

import (
	"io/ioutil"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

func InternalConfig() (*api.OpenShiftManagedCluster, error) {
	var cs api.OpenShiftManagedCluster
	b, err := ioutil.ReadFile("../../_data/containerservice.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(b, &cs)
	return &cs, err
}
