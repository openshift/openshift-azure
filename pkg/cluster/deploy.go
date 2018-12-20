package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
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
	ssHashes, err := u.hashScaleSets(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepHashScaleSets}
	}
	err = u.initializeUpdateBlob(cs, ssHashes)
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
type hash string

func hashVMSS(vmss *compute.VirtualMachineScaleSet) (hash, error) {
	// cleanup capacity so that no unnecessary VM rotations are going to occur
	// because of a scale up/down.
	if vmss.Sku != nil {
		vmss.Sku.Capacity = nil
	}

	data, err := json.Marshal(vmss)
	if err != nil {
		return "", err
	}

	hf := sha256.New()
	hf.Write(data)

	return hash(base64.StdEncoding.EncodeToString(hf.Sum(nil))), nil
}

// hashScaleSets returns the set of desired state scale set hashes
func (u *simpleUpgrader) hashScaleSets(cs *api.OpenShiftManagedCluster) (map[scalesetName]hash, error) {
	ssHashes := map[scalesetName]hash{}

	for _, app := range cs.Properties.AgentPoolProfiles {
		vmss, err := arm.Vmss(&u.pluginConfig, cs, &app, "") // TODO: backupBlob is rather a layering violation here
		if err != nil {
			return nil, err
		}

		h, err := hashVMSS(vmss)
		if err != nil {
			return nil, err
		}

		ssHashes[scalesetName(*vmss.Name)] = h
	}

	return ssHashes, nil
}

func (u *simpleUpgrader) initializeUpdateBlob(cs *api.OpenShiftManagedCluster, ssHashes map[scalesetName]hash) error {
	blob := updateblob{}
	for _, app := range cs.Properties.AgentPoolProfiles {
		for i := 0; i < app.Count; i++ {
			name := instanceName(config.GetInstanceName(app.Name, i))
			blob[name] = ssHashes[scalesetName(config.GetScalesetName(app.Name))]
		}
	}
	return u.writeUpdateBlob(blob)
}
