package v20190430

import (
	"github.com/Azure/go-autorest/autorest/to"
)

func setDefaults(oc *OpenShiftManagedCluster) {
	if oc.Properties == nil {
		oc.Properties = &Properties{}
	}

	if len(oc.Properties.RouterProfiles) == 0 {
		oc.Properties.RouterProfiles = []RouterProfile{
			{
				Name: to.StringPtr("default"),
			},
		}
	}
}
