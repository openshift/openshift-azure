package cluster

import (
	"context"
	"sort"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/config"
)

func (u *simpleUpgrader) WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	return u.kubeclient.WaitForInfraServices(ctx)
}

func (u *simpleUpgrader) waitForNodesInAgentPoolProfile(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, suffix string) error {
	vms, err := u.vmc.List(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(app, suffix), "", "", "")
	if err != nil {
		return err
	}
	for _, vm := range vms {
		computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		u.log.Infof("waiting for %s to be ready", computerName)
		if app.Role == api.AgentPoolProfileRoleMaster {
			err = u.kubeclient.WaitForReadyMaster(ctx, computerName)
		} else {
			err = u.kubeclient.WaitForReadyWorker(ctx, computerName)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// SortedAgentPoolProfilesForRole returns a shallow copy of the
// AgentPoolProfiles of a given role, sorted by name
func SortedAgentPoolProfilesForRole(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (apps []api.AgentPoolProfile) {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			apps = append(apps, app)
		}
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return apps
}

func (u *simpleUpgrader) WaitForMasters(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	for _, app := range SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster) {
		err := u.waitForNodesInAgentPoolProfile(ctx, cs, &app, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *simpleUpgrader) WaitForNodes(ctx context.Context, cs *api.OpenShiftManagedCluster, suffix string) error {
	err := u.WaitForMasters(ctx, cs)
	if err != nil {
		return err
	}

	for _, app := range SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra) {
		err := u.waitForNodesInAgentPoolProfile(ctx, cs, &app, suffix)
		if err != nil {
			return err
		}
	}

	for _, app := range SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute) {
		err := u.waitForNodesInAgentPoolProfile(ctx, cs, &app, suffix)
		if err != nil {
			return err
		}
	}

	return nil
}
