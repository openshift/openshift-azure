package cluster

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// workerScaler implements the logic to scale up and down worker scale sets.  It
// caches the objects associated with the scale set and its VMs to avoid Azure
// API calls where possible.
type workerScaler struct {
	log *logrus.Entry

	ssc        azureclient.VirtualMachineScaleSetsClient
	vmc        azureclient.VirtualMachineScaleSetVMsClient
	kubeclient kubeclient.Kubeclient

	resourceGroup string

	ss    *compute.VirtualMachineScaleSet
	vms   []compute.VirtualMachineScaleSetVM
	vmMap map[string]struct{}
}

func newWorkerScaler(log *logrus.Entry, ssc azureclient.VirtualMachineScaleSetsClient, vmc azureclient.VirtualMachineScaleSetVMsClient, kubeclient kubeclient.Kubeclient, resourceGroup string, ss *compute.VirtualMachineScaleSet) *workerScaler {
	return &workerScaler{log: log, ssc: ssc, vmc: vmc, kubeclient: kubeclient, resourceGroup: resourceGroup, ss: ss}
}

// initializeCache fetches the scale set's VMs from the Azure API and updates
// the cache.
func (ws *workerScaler) initializeCache(ctx context.Context) error {
	vms, err := ws.vmc.List(ctx, ws.resourceGroup, *ws.ss.Name, "", "", "")
	if err != nil {
		return err
	}

	ws.updateCache(vms)

	return nil
}

// updateCache updates the cached list of scale set VMs, ensures that ws.vmMap is
// correct, and ensures that the recorded scale set capacity matches the number
// of VMs.
func (ws *workerScaler) updateCache(vms []compute.VirtualMachineScaleSetVM) {
	ws.vms = vms

	ws.vmMap = make(map[string]struct{}, len(ws.vms))
	for _, vm := range ws.vms {
		ws.vmMap[*vm.InstanceID] = struct{}{}
	}

	ws.ss.Sku.Capacity = to.Int64Ptr(int64(len(vms)))
}

// scale sets the scale set capacity to count.
func (ws *workerScaler) scale(ctx context.Context, count int64) *api.PluginError {
	switch {
	case *ws.ss.Sku.Capacity < count:
		return ws.scaleUp(ctx, count)
	case *ws.ss.Sku.Capacity > count:
		return ws.scaleDown(ctx, count)
	default:
		return nil
	}
}

// scaleUp increases the scale set capacity to count.  It detects newly created
// instances and waits for them to become ready.
func (ws *workerScaler) scaleUp(ctx context.Context, count int64) *api.PluginError {
	ws.log.Infof("scaling %s capacity up from %d to %d", *ws.ss.Name, *ws.ss.Sku.Capacity, count)

	if ws.vms == nil {
		if err := ws.initializeCache(ctx); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolListVMs}
		}
	}

	err := ws.ssc.Update(ctx, ws.resourceGroup, *ws.ss.Name, compute.VirtualMachineScaleSetUpdate{
		Sku: &compute.Sku{
			Capacity: to.Int64Ptr(count),
		},
	})
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolUpdateScaleSet}
	}

	vms, err := ws.vmc.List(ctx, ws.resourceGroup, *ws.ss.Name, "", "", "")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolListVMs}
	}

	for _, vm := range vms {
		if _, found := ws.vmMap[*vm.InstanceID]; !found {
			computerName := *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
			ws.log.Infof("waiting for %s to be ready", computerName)

			err = ws.kubeclient.WaitForReadyWorker(ctx, kubeclient.ComputerName(computerName))
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolWaitForReady}
			}
		}
	}

	ws.updateCache(vms)

	return nil
}

// scaleDown decreases the scale set capacity to count by individually deleting
// excess instances.
func (ws *workerScaler) scaleDown(ctx context.Context, count int64) *api.PluginError {
	ws.log.Infof("scaling %s capacity down from %d to %d", *ws.ss.Name, *ws.ss.Sku.Capacity, count)

	if ws.vms == nil {
		if err := ws.initializeCache(ctx); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolListVMs}
		}
	}

	for _, vm := range ws.vms[count:] {
		computerName := *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName

		// TODO: should probably mark all appropriate nodes unschedulable, then
		// do the draining, then do the deleting in parallel.
		ws.log.Infof("draining %s", computerName)
		if err := ws.kubeclient.DrainAndDeleteWorker(ctx, kubeclient.ComputerName(computerName)); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolDrain}
		}

		ws.log.Infof("deleting %s", computerName)
		err := ws.vmc.Delete(ctx, ws.resourceGroup, *ws.ss.Name, *vm.InstanceID)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolDeleteVM}
		}
	}

	ws.updateCache(ws.vms[:count])

	return nil
}
