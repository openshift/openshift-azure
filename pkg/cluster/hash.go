package cluster

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
)

func hashVMSS(vmss *compute.VirtualMachineScaleSet) ([]byte, error) {
	// cleanup capacity so that no unnecessary VM rotations are going to occur
	// because of a scale up/down.
	if vmss.Sku != nil {
		vmss.Sku.Capacity = nil
	}

	data, err := json.Marshal(vmss)
	if err != nil {
		return nil, err
	}

	hf := sha256.New()
	hf.Write(data)

	return hf.Sum(nil), nil
}

// hashScaleSets returns the set of desired state scale set hashes
func (u *simpleUpgrader) hashScaleSets(cs *api.OpenShiftManagedCluster) (map[scalesetName][]byte, error) {
	ssHashes := map[scalesetName][]byte{}

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
