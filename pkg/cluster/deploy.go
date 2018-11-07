package cluster

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) error {
	err := u.createClients(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}
	err = deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	err = u.Initialize(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInitialize}
	}
	err = u.initializeUpdateBlob(ctx, cs, azuretemplate)
	if err != nil {
		log.Warnf("could not initialize update blob: %v", err)
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

func (u *simpleUpgrader) initializeUpdateBlob(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}) error {
	blob := make(map[string]string)
	for _, r := range jsonpath.MustCompile("$.resources.*").Get(azuretemplate) {
		resource, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if !isScaleSet(resource) {
			continue
		}

		// cleanup capacity so that no unnecessary VM rotations are going
		// to occur because of a scale up/down.
		jsonpath.MustCompile("$.sku.capacity").Delete(resource)

		// hash scale set
		data, err := json.Marshal(resource)
		if err != nil {
			return err
		}
		hf := sha256.New()
		fmt.Fprintf(hf, string(data))

		blob[getRole(resource)] = base64.StdEncoding.EncodeToString(hf.Sum(nil))
	}

	for role, hash := range blob {
		vms, err := listVMs(ctx, cs, u.vmc, api.AgentPoolProfileRole(role))
		if err != nil {
			return err
		}
		for _, vm := range vms {
			blob[*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName] = hash
		}
	}

	return u.updateBlob(blob)
}

func isScaleSet(resource map[string]interface{}) bool {
	for k, v := range resource {
		if k == "type" && v.(string) == "Microsoft.Compute/virtualMachineScaleSets" {
			return true
		}
	}
	return false
}

func getRole(resource map[string]interface{}) string {
	for k, v := range resource {
		if k == "name" && strings.HasPrefix(v.(string), "ss-") {
			return v.(string)[3:]
		}
	}
	return ""
}

func (u *simpleUpgrader) updateBlob(blob map[string]string) error {
	data, err := json.Marshal(blob)
	if err != nil {
		return err
	}
	bsc := u.storageClient.GetBlobService()
	c := bsc.GetContainerReference("update")
	b := c.GetBlobReference("update")
	return b.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
}
