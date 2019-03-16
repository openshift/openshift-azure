package cluster

import (
	"context"
	"sort"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
)

func (u *simpleUpgrader) WaitForNodesInAgentPoolProfile(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, suffix string) error {
	vms, err := u.vmc.List(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(app, suffix), "", "", "")
	if err != nil {
		return err
	}
	for _, vm := range vms {
		hostname := strings.ToLower(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		u.log.Infof("waiting for %s to be ready", hostname)
		if app.Role == api.AgentPoolProfileRoleMaster {
			err = u.Kubeclient.WaitForReadyMaster(ctx, hostname)
		} else {
			err = u.Kubeclient.WaitForReadyWorker(ctx, hostname)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// SortedAgentPoolProfilesForRole returns a shallow copy of the
// AgentPoolProfiles of a given role, sorted by name
func (u *simpleUpgrader) SortedAgentPoolProfilesForRole(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (apps []api.AgentPoolProfile) {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			apps = append(apps, app)
		}
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return apps
}
