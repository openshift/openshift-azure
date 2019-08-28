package v20190930preview

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

	if len(oc.Properties.AgentPoolProfiles) == 0 {
		oc.Properties.AgentPoolProfiles = []AgentPoolProfile{
			{
				Name: to.StringPtr("infra"),
				Role: (*AgentPoolProfileRole)(to.StringPtr("infra")),
			},
		}
	}

	for i := range oc.Properties.AgentPoolProfiles {
		if oc.Properties.AgentPoolProfiles[i].Name != nil &&
			*oc.Properties.AgentPoolProfiles[i].Name == "infra" &&
			oc.Properties.AgentPoolProfiles[i].Count == nil {
			oc.Properties.AgentPoolProfiles[i].Count = to.Int64Ptr(3)
		}

		if oc.Properties.AgentPoolProfiles[i].OSType == nil {
			oc.Properties.AgentPoolProfiles[i].OSType = (*OSType)(to.StringPtr("Linux"))
		}
	}

	if len(oc.Properties.RouterProfiles) == 0 {
		oc.Properties.RouterProfiles = []RouterProfile{
			{
				Name: to.StringPtr("default"),
			},
		}
	}

	if oc.Properties.NetworkProfile == nil {
		oc.Properties.NetworkProfile = &NetworkProfile{}
	}

	if oc.Properties.NetworkProfile.VnetCIDR == nil {
		oc.Properties.NetworkProfile.VnetCIDR = to.StringPtr("10.0.0.0/8")
	}
	if oc.Properties.NetworkProfile.DefaultCIDR == nil {
		oc.Properties.NetworkProfile.DefaultCIDR = to.StringPtr("10.0.0.0/24")
	}
	if oc.Properties.NetworkProfile.ManagementCIDR == nil {
		oc.Properties.NetworkProfile.ManagementCIDR = to.StringPtr("10.0.2.0/24")
	}
}
