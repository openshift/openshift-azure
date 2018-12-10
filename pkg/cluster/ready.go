package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
)

func (u *simpleUpgrader) WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	return u.kubeclient.WaitForInfraServices(ctx)
}

func (u *simpleUpgrader) waitForNodes(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	for _, role := range []api.AgentPoolProfileRole{api.AgentPoolProfileRoleMaster, api.AgentPoolProfileRoleInfra, api.AgentPoolProfileRoleCompute} {
		vms, err := u.listVMs(ctx, cs, role)
		if err != nil {
			return err
		}
		for _, vm := range vms {
			computerName := kubeclient.ComputerName(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			u.log.Infof("waiting for %s to be ready", computerName)
			err = u.kubeclient.WaitForReady(ctx, role, computerName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
