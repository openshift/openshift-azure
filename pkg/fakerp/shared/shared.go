package shared

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/openshift/openshift-azure/pkg/api"
)

const (
	DataDirectory = "_data/"
	LocalHttpAddr = "localhost:8080"
)

// IsUpdate return whether or not this is an update or create.
func IsUpdate() bool {
	_, err := os.Stat("_data/containerservice.yaml")
	return err == nil
}

// DiscoverInternalConfig discover and returns the internal config struct
func DiscoverInternalConfig() (*api.OpenShiftManagedCluster, error) {
	dataDir, err := FindDirectory(DataDirectory)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(filepath.Join(dataDir, "containerservice.yaml"))
	if err != nil {
		return nil, err
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, err
	}

	return cs, nil
}
