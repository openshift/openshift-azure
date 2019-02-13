package shared

import (
	"os"
	"path/filepath"
	"reflect"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

const (
	AdminContext  = "/admin"
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

// IsScaleOperation return whether or not this is a scale up/down operation
func IsScaleOperation(cs, oldCs *api.OpenShiftManagedCluster) bool {
	var agentCountChanged bool
	for i, pool := range cs.Properties.AgentPoolProfiles {
		if pool.Count != oldCs.Properties.AgentPoolProfiles[i].Count {
			agentCountChanged = agentCountChanged || true
		}
	}
	if !agentCountChanged {
		return false
	}
	deepCs := cs.DeepCopy()
	for i, pool := range oldCs.Properties.AgentPoolProfiles {
		deepCs.Properties.AgentPoolProfiles[i].Count = pool.Count
	}
	return reflect.DeepEqual(deepCs, oldCs)
}

// DiscoverInternalConfig discover and returns the internal config struct
func DiscoverInternalConfig() (*api.OpenShiftManagedCluster, error) {
	dataDir, err := FindDirectory(DataDirectory)
	if err != nil {
		return nil, err
	}
	return managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
}
