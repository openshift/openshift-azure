package shared

import (
	"os"
	"path/filepath"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

const (
	DataDirectory = "_data/"
	LocalHttpAddr = "localhost:8080"
)

// IsUpdate return whether or not this is an update or create.
func IsUpdate() bool {
	dataDir, err := FindDirectory(DataDirectory)
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dataDir, "containerservice.yaml")); err == nil {
		return true
	}
	return false
}

// DiscoverInternalConfig discover and returns the internal config struct
func DiscoverInternalConfig() (*api.OpenShiftManagedCluster, error) {
	dataDir, err := FindDirectory(DataDirectory)
	if err != nil {
		return nil, err
	}
	return managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
}
