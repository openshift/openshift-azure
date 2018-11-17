package cluster

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) *api.PluginError {
	err := u.createClients(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}
	err = deployFn(ctx, azuretemplate)
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
	err = managedcluster.WaitForHealthz(ctx, cs.Config.AdminKubeconfig)
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

type vmInfo struct {
	InstanceName instanceName `json:"instanceName,omitempty"`
	ScalesetHash hash         `json:"scalesetHash,omitempty"`
}

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
	vmHashes := make(map[instanceName]hash)
	for _, profile := range cs.Properties.AgentPoolProfiles {
		for i := 0; i < profile.Count; i++ {
			name := instanceName(fmt.Sprintf("ss-%s_%d", profile.Name, i))
			vmHashes[name] = ssHashes[scalesetName("ss-"+profile.Name)]
		}
	}
	return u.updateBlob(vmHashes)
}

const updateContainerName = "update"
const updateBlobName = "update"

func (u *simpleUpgrader) updateBlob(b map[instanceName]hash) error {
	blob := make([]vmInfo, len(b))
	for instancename, hash := range b {
		blob = append(blob, vmInfo{
			InstanceName: instancename,
			ScalesetHash: hash,
		})
	}
	data, err := json.Marshal(blob)
	if err != nil {
		return err
	}
	bsc := u.storageClient.GetBlobService()
	c := bsc.GetContainerReference(updateContainerName)
	bc := c.GetBlobReference(updateBlobName)
	return bc.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
}

func (u *simpleUpgrader) readBlob() (map[instanceName]hash, error) {
	bsc := u.storageClient.GetBlobService()
	c := bsc.GetContainerReference(updateContainerName)
	bc := c.GetBlobReference(updateBlobName)

	rc, err := bc.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var blob []vmInfo
	if err := json.Unmarshal(data, &blob); err != nil {
		return nil, err
	}
	b := make(map[instanceName]hash)
	for _, vi := range blob {
		b[vi.InstanceName] = vi.ScalesetHash
	}
	return b, nil
}
