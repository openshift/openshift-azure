package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
)

func (u *simpleUpgrader) WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	return u.kubeclient.WaitForInfraServices(ctx)
}

func (u *simpleUpgrader) waitForNode(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) error {
	vms, err := u.listVMs(ctx, cs, app)
	if err != nil {
		return err
	}
	for _, vm := range vms {
		computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
		u.log.Infof("waiting for %s to be ready", computerName)
		err = u.kubeclient.WaitForReady(ctx, app.Role, computerName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *simpleUpgrader) waitForNodes(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster) {
		err := u.waitForNode(ctx, cs, &app)
		if err != nil {
			return err
		}
	}

	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra) {
		err := u.waitForNode(ctx, cs, &app)
		if err != nil {
			return err
		}
	}

	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute) {
		err := u.waitForNode(ctx, cs, &app)
		if err != nil {
			return err
		}
	}

	return nil
}
