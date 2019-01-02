package cluster

import (
	"context"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"

	"github.com/openshift/openshift-azure/pkg/api"
)

func (u *simpleUpgrader) Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn, suffix string) *api.PluginError {
	err := deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	err = u.initialize(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitialize}
	}
	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster) {
		if perr := u.updateMasterAgentPool(ctx, cs, &app); perr != nil {
			return perr
		}
	}
	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra) {
		if perr := u.updateWorkerAgentPool(ctx, cs, &app, suffix); perr != nil {
			return perr
		}
	}
	for _, app := range sortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute) {
		if perr := u.updateWorkerAgentPool(ctx, cs, &app, suffix); perr != nil {
			return perr
		}
	}
	return nil
}

func (u *simpleUpgrader) listScalesets(ctx context.Context, resourceGroup, scalesetPrefix string) ([]compute.VirtualMachineScaleSet, error) {
	ssPages, err := u.ssc.List(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}

	var scalesets []compute.VirtualMachineScaleSet
	for ssPages.NotDone() {
		for _, ss := range ssPages.Values() {
			if strings.HasPrefix(*ss.Name, scalesetPrefix) {
				scalesets = append(scalesets, ss)
			}
		}

		err = ssPages.Next()
		if err != nil {
			return nil, err
		}
	}

	return scalesets, nil
}

// sortedAgentPoolProfilesForRole returns a shallow copy of the
// AgentPoolProfiles of a given role, sorted by name
func sortedAgentPoolProfilesForRole(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) (apps []api.AgentPoolProfile) {
	for _, app := range cs.Properties.AgentPoolProfiles {
		if app.Role == role {
			apps = append(apps, app)
		}
	}

	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })

	return apps
}
