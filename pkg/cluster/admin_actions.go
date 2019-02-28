package cluster

import (
	"context"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
)

func (u *simpleUpgrader) Reimage(ctx context.Context, cs *api.OpenShiftManagedCluster, scaleset, instanceID string) error {
	return u.vmc.Reimage(ctx, cs.Properties.AzProfile.ResourceGroup, scaleset, instanceID, nil)
}

func (u *simpleUpgrader) ListVMHostnames(ctx context.Context, cs *api.OpenShiftManagedCluster) ([]string, error) {
	scalesets, err := u.ssc.List(ctx, cs.Properties.AzProfile.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var hostnames []string
	for _, ss := range scalesets {
		vms, err := u.vmc.List(ctx, cs.Properties.AzProfile.ResourceGroup, *ss.Name, "", "", "")
		if err != nil {
			return nil, err
		}

		for _, vm := range vms {
			hostnames = append(hostnames, strings.ToLower(*vm.OsProfile.ComputerName))
		}
	}

	return hostnames, nil
}
