package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Evacuate(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	// We may need/want to delete all the scalesets in the future
	future, err := u.ssc.Delete(ctx, cs.Properties.AzProfile.ResourceGroup, "ss-master")
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
	ssHashes, err := hashScaleSets(azuretemplate)
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

func hashScaleSets(azuretemplate map[string]interface{}) (map[scalesetName]hash, error) {
	ssHashes := make(map[scalesetName]hash)
	for _, r := range jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Compute/virtualMachineScaleSets')]").Get(azuretemplate) {
		original, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		// deep-copy the ARM template since we are mutating it below.
		resource := deepCopy(original)

		// cleanup capacity so that no unnecessary VM rotations are going
		// to occur because of a scale up/down.
		jsonpath.MustCompile("$.sku.capacity").Delete(resource)

		// filter out the nsg dependsOn entry since we remove it
		// during upgrades due to an azure issue.
		jsonpath.MustCompile("$.dependsOn").Delete(resource)

		// hash scale set
		data, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		hf := sha256.New()
		hf.Write(data)

		scaleSetName := jsonpath.MustCompile("$.name").MustGetString(resource)
		ssHashes[scalesetName(scaleSetName)] = hash(base64.StdEncoding.EncodeToString(hf.Sum(nil)))
	}
	return ssHashes, nil
}

func deepCopy(in map[string]interface{}) map[string]interface{} {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	var out map[string]interface{}
	err = json.Unmarshal(b, &out)
	if err != nil {
		panic(err)
	}
	return out
}

func (u *simpleUpgrader) initializeUpdateBlob(cs *api.OpenShiftManagedCluster, ssHashes map[scalesetName]hash) error {
	blob := updateblob{}
	for _, profile := range cs.Properties.AgentPoolProfiles {
		for i := 0; i < profile.Count; i++ {
			name := instanceName(fmt.Sprintf("ss-%s_%d", profile.Name, i))
			blob[name] = ssHashes[scalesetName("ss-"+profile.Name)]
		}
	}
	return u.writeUpdateBlob(blob)
}
