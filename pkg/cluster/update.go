package cluster

import (
	"context"
	"sort"

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
