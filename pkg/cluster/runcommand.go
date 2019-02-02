package cluster

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
)

func (u *simpleUpgrader) RunCommand(ctx context.Context, oc *api.OpenShiftManagedCluster, scaleSetName, instanceId string, parameters compute.RunCommandInput) (compute.RunCommandResult, *api.PluginError) {
	result, err := u.vmc.RunCommand(ctx, oc.Properties.AzProfile.ResourceGroup, scaleSetName, instanceId, parameters)
	if err != nil {
		return result, &api.PluginError{Err: err, Step: "RunGenericCommand"}
	}
	return result, nil
}

func (u *simpleUpgrader) RestartDocker(ctx context.Context, oc *api.OpenShiftManagedCluster, scaleSetName, instanceId string) (compute.RunCommandResult, *api.PluginError) {
	params := compute.RunCommandInput{
		CommandID: to.StringPtr("RunShellScript"),
		Script:    to.StringSlicePtr([]string{"systemctl restart docker.service"}),
	}
	return u.RunCommand(ctx, oc, scaleSetName, instanceId, params)
}

func (u *simpleUpgrader) RestartKubelet(ctx context.Context, oc *api.OpenShiftManagedCluster, scaleSetName, instanceId string) (compute.RunCommandResult, *api.PluginError) {
	params := compute.RunCommandInput{
		CommandID: to.StringPtr("RunShellScript"),
		Script:    to.StringSlicePtr([]string{"systemctl restart atomic-openshift-node.service"}),
	}
	return u.RunCommand(ctx, oc, scaleSetName, instanceId, params)
}

func (u *simpleUpgrader) RestartNetworkManager(ctx context.Context, oc *api.OpenShiftManagedCluster, scaleSetName, instanceId string) (compute.RunCommandResult, *api.PluginError) {
	params := compute.RunCommandInput{
		CommandID: to.StringPtr("RunShellScript"),
		Script:    to.StringSlicePtr([]string{"systemctl restart NetworkManager.service"}),
	}
	return u.RunCommand(ctx, oc, scaleSetName, instanceId, params)
}
