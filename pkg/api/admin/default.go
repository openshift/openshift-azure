package admin

import (
	"github.com/openshift/openshift-azure/pkg/api"
)

func setDefaults(cs *api.OpenShiftManagedCluster) {
	// upgrade v7 to v8+ requires additional network fields to be added.
	// TODO: Remove when v7 is gone
	if len(cs.Properties.NetworkProfile.VnetCIDR) == 0 {
		cs.Properties.NetworkProfile.VnetCIDR = "10.0.0.0/8"
	}
	if len(cs.Properties.NetworkProfile.DefaultCIDR) == 0 {
		cs.Properties.NetworkProfile.DefaultCIDR = "10.0.0.0/24"
	}
	if len(cs.Properties.NetworkProfile.ManagementCIDR) == 0 {
		cs.Properties.NetworkProfile.ManagementCIDR = "10.0.2.0/24"
	}
}
