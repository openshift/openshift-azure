package scaler

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/scaler.go -package=mock_$GOPACKAGE  -source scaler.go
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/scaler.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/scaler.go

import (
	"context"
	"crypto/rsa"
	"strings"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/compute"
	"github.com/openshift/openshift-azure/test/util/diagnostics"
)

// Factory interface is to create new Scaler instances.
type Factory interface {
	New(log *logrus.Entry, ssc compute.VirtualMachineScaleSetsClient, vmc compute.VirtualMachineScaleSetVMsClient, kubeclient kubeclient.Interface, resourceGroup string, ss *azcompute.VirtualMachineScaleSet, testConfig api.TestConfig, sshkey *rsa.PrivateKey) Scaler
}

type scalerFactory struct{}

var _ Factory = &scalerFactory{}

// NewFactory create a new Factory instance.
func NewFactory() Factory {
	return &scalerFactory{}
}

// Scaler is an interface that changes the number of VMs in a scaleset until they meet
// the 'count' argument. Scale function simply checks which direction
// to scale up or down.
type Scaler interface {
	Scale(ctx context.Context, count int64) *api.PluginError
}

// workerScaler implements the logic to scale up and down worker scale sets.  It
// caches the objects associated with the scale set and its VMs to avoid Azure
// API calls where possible.
type workerScaler struct {
	log *logrus.Entry

	ssc        compute.VirtualMachineScaleSetsClient
	vmc        compute.VirtualMachineScaleSetVMsClient
	kubeclient kubeclient.Interface

	resourceGroup string

	ss    *azcompute.VirtualMachineScaleSet
	vms   []azcompute.VirtualMachineScaleSetVM
	vmMap map[string]struct{}

	testConfig api.TestConfig
	sshkey     *rsa.PrivateKey
}

var _ Scaler = &workerScaler{}

func (sf *scalerFactory) New(log *logrus.Entry, ssc compute.VirtualMachineScaleSetsClient, vmc compute.VirtualMachineScaleSetVMsClient, kubeclient kubeclient.Interface, resourceGroup string, ss *azcompute.VirtualMachineScaleSet, testConfig api.TestConfig, sshkey *rsa.PrivateKey) Scaler {
	return &workerScaler{log: log, ssc: ssc, vmc: vmc, kubeclient: kubeclient, resourceGroup: resourceGroup, ss: ss, testConfig: testConfig, sshkey: sshkey}
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
func (ws *workerScaler) updateCache(vms []azcompute.VirtualMachineScaleSetVM) {
	ws.vms = vms

	ws.vmMap = make(map[string]struct{}, len(ws.vms))
	for _, vm := range ws.vms {
		ws.vmMap[*vm.InstanceID] = struct{}{}
	}

	ws.ss.Sku.Capacity = to.Int64Ptr(int64(len(vms)))
}

// Scale sets the scale set capacity to count.
func (ws *workerScaler) Scale(ctx context.Context, count int64) *api.PluginError {
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

	err := ws.ssc.Update(ctx, ws.resourceGroup, *ws.ss.Name, azcompute.VirtualMachineScaleSetUpdate{
		Sku: &azcompute.Sku{
			Capacity: to.Int64Ptr(count),
		},
	})
	if err != nil && ws.testConfig.RunningUnderTest && ws.testConfig.ArtifactDir != "" {
		// this is tempory whilst investigating: https://github.com/openshift/openshift-azure/issues/1410
		subscriptionID := ""
		if ws.ss.ID != nil {
			subscriptionID = strings.Split(*ws.ss.ID, "/")[2]
		}
		ssd := diagnostics.NewScalesetDebugger(ws.log, subscriptionID, ws.resourceGroup, ws.testConfig, ws.ssc)
		ssd.GatherStatuses(ctx, *ws.ss.Name)
		ssd.GatherActivityLogs(ctx)
		if ws.sshkey != nil {
			ssd.GatherHostLogs(ctx, *ws.ss.Name, count, ws.sshkey)
		}
	}

	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolUpdateScaleSet}
	}

	vms, err := ws.vmc.List(ctx, ws.resourceGroup, *ws.ss.Name, "", "", "")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolListVMs}
	}

	for _, vm := range vms {
		if _, found := ws.vmMap[*vm.InstanceID]; !found {
			hostname := strings.ToLower(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)
			ws.log.Infof("waiting for %s to be ready", hostname)

			err = ws.kubeclient.WaitForReadyWorker(ctx, hostname)
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
		hostname := strings.ToLower(*vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName)

		// TODO: should probably mark all appropriate nodes unschedulable, then
		// do the draining, then do the deleting in parallel.
		ws.log.Infof("draining %s", hostname)
		if err := ws.kubeclient.DrainAndDeleteWorker(ctx, hostname); err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolDrain}
		}

		ws.log.Infof("deleting %s", hostname)
		err := ws.vmc.Delete(ctx, ws.resourceGroup, *ws.ss.Name, *vm.InstanceID)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateWorkerAgentPoolDeleteVM}
		}
	}

	ws.updateCache(ws.vms[:count])

	return nil
}
