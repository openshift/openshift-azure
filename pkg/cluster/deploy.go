package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn, suffix string) *api.PluginError {
	err := deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	err = u.initialize(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitialize}
	}
	err = u.initializeUpdateBlob(cs, suffix)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
	}
	_, err = wait.ForHTTPStatusOk(ctx, u.log, u.rt, "https://"+cs.Properties.FQDN+"/healthz")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	err = u.waitForNodes(ctx, cs, suffix)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
	}
	return nil
}

func (u *simpleUpgrader) initializeUpdateBlob(cs *api.OpenShiftManagedCluster, suffix string) error {
	blob := updateblob.NewUpdateBlob()
	for _, app := range cs.Properties.AgentPoolProfiles {
		h, err := u.hasher.HashScaleSet(cs, &app)
		if err != nil {
			return err
		}
		if app.Role == api.AgentPoolProfileRoleMaster {
			for i := int64(0); i < app.Count; i++ {
				name := config.GetMasterInstanceName(i)
				blob.InstanceHashes[name] = h
			}
		} else {
			blob.ScalesetHashes[config.GetScalesetName(&app, suffix)] = h
		}
	}
	return u.updateBlobService.Write(blob)
}
