package v20190430

import (
	"github.com/Azure/go-autorest/autorest/to"
)

func setDefaults(oc *OpenShiftManagedCluster) {
	if oc.Properties == nil {
		oc.Properties = &Properties{}
	}

	if oc.Properties.MasterPoolProfile == nil {
		oc.Properties.MasterPoolProfile = &MasterPoolProfile{}
	}

	if oc.Properties.MasterPoolProfile.Count == nil {
		oc.Properties.MasterPoolProfile.Count = to.Int64Ptr(3)
	}

	if len(oc.Properties.RouterProfiles) == 0 {
		oc.Properties.RouterProfiles = []RouterProfile{
			{
				Name: to.StringPtr("default"),
			},
		}
	}
}
