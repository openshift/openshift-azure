package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Evacuate(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	// We may need/want to delete all the scalesets in the future
	future, err := u.ssc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, config.GetScalesetName(string(api.AgentPoolProfileRoleMaster)))
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepScaleSetDelete}
	}
	err = future.WaitForCompletionRef(ctx, u.ssc.Client())
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepScaleSetDelete}
	}
	if u.storageClient == nil {
		keys, err := u.accountsClient.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
		}
		u.storageClient, err = storage.NewClient(cs.Config.ConfigStorageAccount, *(*keys.Keys)[0].Value, storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
		}
	}
	err = u.deleteUpdateBlob()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeleteBlob}
	}
	return nil
}

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) *api.PluginError {
	err := deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	err = u.initialize(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitialize}
	}
	err = u.initializeUpdateBlob(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
	}
	err = managedcluster.WaitForHealthz(ctx, u.log, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	err = u.waitForNodes(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
	}
	return nil
}

type scalesetName string
type instanceName string

func (u *simpleUpgrader) initializeUpdateBlob(cs *api.OpenShiftManagedCluster) error {
	blob := newUpdateBlob()
	for _, app := range cs.Properties.AgentPoolProfiles {
		h, err := u.hasher.HashScaleSet(cs, &app)
		if err != nil {
			return err
		}
		for i := int64(0); i < app.Count; i++ {
			name := instanceName(config.GetInstanceName(app.Name, int(i)))
			blob.InstanceHashes[name] = h
		}
	}
	return u.writeUpdateBlob(blob)
}
