package features

import (
	"github.com/openshift/openshift-azure/pkg/api"
)

// PrivateLinkEnabled checks if PrivateLink is/should be enabled on the cluster.
// It is set
func PrivateLinkEnabled(cs *api.OpenShiftManagedCluster) bool {
	if cs.Properties.NetworkProfile.ManagementSubnetCIDR != nil {
		return true
	}
	return false
}
