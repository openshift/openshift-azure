package v20180930preview

import (
	"github.com/Azure/go-autorest/autorest/to"
)

func setDefaults(oc *OpenShiftManagedCluster) {
	if len(oc.Properties.RouterProfiles) == 0 {
		oc.Properties.RouterProfiles = []RouterProfile{
			{
				Name: to.StringPtr("default"),
			},
		}
	}
}
