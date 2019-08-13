package runtime

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/sirupsen/logrus"
)

// Versions contains all supported versions
var Versions map[string]func(context.Context, *logrus.Entry, *api.OpenShiftManagedCluster, api.TestConfig) api.ARMInterface

// AddToVersion adds version to versions slice
func AddToVersion(version string, f func(context.Context, *logrus.Entry, *api.OpenShiftManagedCluster, api.TestConfig) api.ARMInterface) {
	if Versions == nil {
		Versions = make(map[string]func(context.Context, *logrus.Entry, *api.OpenShiftManagedCluster, api.TestConfig) api.ARMInterface)
	}
	Versions[version] = f
}
