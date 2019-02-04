package cluster

import (
	"context"
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
)

// VmIdentifier represents an aggregation of the hostname, instance id, resource id and scale set identifiers for an azure vm
type VmIdentifier struct {
	Hostname   kubeclient.ComputerName `json:"computerName,omitempty"`
	InstanceId string                  `json:"instanceId,omitempty"`
	ResourceId string                  `json:"resourceId,omitempty"`
	ScaleSet   string                  `json:"scaleSetName,omitempty"`
}

func (u *simpleUpgrader) ListHostnames(ctx context.Context, resourceGroup string) ([]*VmIdentifier, error) {
	scaleSets, err := u.ssc.List(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}
	var names []*VmIdentifier
	for _, scaleSet := range scaleSets {
		scalesetName := *scaleSet.Name
		vms, err := u.vmc.List(ctx, resourceGroup, scalesetName, "", "", "")
		if err != nil {
			return nil, err
		}
		for _, vm := range vms {
			name := &VmIdentifier{
				Hostname:   kubeclient.ComputerName(*vm.OsProfile.ComputerName),
				InstanceId: *vm.InstanceID,
				ResourceId: *vm.ID,
				ScaleSet:   scalesetName,
			}
			names = append(names, name)
		}
	}
	return names, nil
}

func (u *simpleUpgrader) VmIdByHostname(ctx context.Context, resourceGroup string, hostname kubeclient.ComputerName) (*VmIdentifier, error) {
	vms, err := u.ListHostnames(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}
	var vmId *VmIdentifier
	for _, vm := range vms {
		if vm.Hostname == hostname {
			vmId = vm
			break
		}
	}
	if vmId == nil {
		return nil, fmt.Errorf("%s not found", hostname)
	}
	return vmId, nil
}

func (u *simpleUpgrader) Reimage(ctx context.Context, cs *api.OpenShiftManagedCluster, hostname kubeclient.ComputerName) error {
	vmId, err := u.VmIdByHostname(ctx, cs.Properties.AzProfile.ResourceGroup, hostname)
	if err != nil {
		return err
	}
	err = u.vmc.Reimage(ctx, cs.Properties.AzProfile.ResourceGroup, vmId.ScaleSet, vmId.InstanceId, nil)
	return err
}
