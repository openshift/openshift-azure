package compute

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/compute"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
)

type ScaleSetState struct {
	Name   string
	Vms    []azcompute.VirtualMachineScaleSetVM
	VmsDir map[string]string
}

type ComputeRP struct {
	Log   *logrus.Entry
	Calls []string
	Cs    *api.OpenShiftManagedCluster
	Ssc   []azcompute.VirtualMachineScaleSet
	State []ScaleSetState
}

type FakeVirtualMachineScaleSetVMsClient struct {
	rp  *ComputeRP
	kvc keyvault.KeyVaultClient
}

var _ compute.VirtualMachineScaleSetVMsClient = &FakeVirtualMachineScaleSetVMsClient{}

func (c *ComputeRP) getScaleSetStateIndex(VMScaleSetName string) int {
	for i, sss := range c.State {
		if sss.Name == VMScaleSetName {
			return i
		}
	}
	return -1
}

func (c *ComputeRP) createVM(ss *azcompute.VirtualMachineScaleSet, stateIndex, vmIndex int) string {
	name := fmt.Sprintf("%s-%d", (*ss.Name)[3:], vmIndex)
	compName := fmt.Sprintf("%s-%06d", (*ss.Name)[3:], vmIndex)
	vm := azcompute.VirtualMachineScaleSetVM{
		ID:         to.StringPtr(uuid.NewV4().String()),
		InstanceID: to.StringPtr(fmt.Sprintf("%d", vmIndex)),
		Name:       &name,
		VirtualMachineScaleSetVMProperties: &azcompute.VirtualMachineScaleSetVMProperties{
			OsProfile: &azcompute.OSProfile{ComputerName: &compName},
		},
		Location: ss.Location,
		Plan:     ss.Plan,
		Tags:     ss.Tags,
		Zones:    ss.Zones,
	}
	c.State[stateIndex].Vms = append(c.State[stateIndex].Vms, vm)
	return *vm.InstanceID
}

// NewFakeVirtualMachineScaleSetVMsClient creates a new Fake instance
func NewFakeVirtualMachineScaleSetVMsClient(kvc keyvault.KeyVaultClient, rp *ComputeRP) *FakeVirtualMachineScaleSetVMsClient {
	return &FakeVirtualMachineScaleSetVMsClient{rp: rp, kvc: kvc}
}

// Deallocate Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Deallocate:"+VMScaleSetName+":"+instanceID)
	stateIndex := v.rp.getScaleSetStateIndex(VMScaleSetName)
	if stateIndex == -1 {
		return fmt.Errorf("ScaleSet %s not found", VMScaleSetName)
	}
	for _, vm := range v.rp.State[stateIndex].Vms {
		if *vm.InstanceID == instanceID {
			v.rp.Log.Debugf("deallocating VM %s:%s", VMScaleSetName, instanceID)
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}

// Delete Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Delete:"+VMScaleSetName+":"+instanceID)
	stateIndex := v.rp.getScaleSetStateIndex(VMScaleSetName)
	if stateIndex == -1 {
		return fmt.Errorf("ScaleSet %s not found", VMScaleSetName)
	}
	for s, vm := range v.rp.State[stateIndex].Vms {
		if *vm.InstanceID == instanceID {
			v.rp.Log.Debugf("deleting VM %s:%s", VMScaleSetName, instanceID)
			os.RemoveAll(v.rp.State[stateIndex].VmsDir[instanceID])
			delete(v.rp.State[stateIndex].VmsDir, instanceID)
			v.rp.State[stateIndex].Vms = append(v.rp.State[stateIndex].Vms[:s], v.rp.State[stateIndex].Vms[s+1:]...)
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}

// List Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName, VMScaleSetName, filter, selectParameter, expand string) ([]azcompute.VirtualMachineScaleSetVM, error) {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:List:"+VMScaleSetName)
	stateIndex := v.rp.getScaleSetStateIndex(VMScaleSetName)
	if stateIndex == -1 {
		return nil, fmt.Errorf("ScaleSet %s not found", VMScaleSetName)
	}
	return v.rp.State[stateIndex].Vms, nil
}

// Reimage Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Reimage(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string, VMScaleSetVMReimageInput *azcompute.VirtualMachineScaleSetVMReimageParameters) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Reimage:"+VMScaleSetName+":"+instanceID)
	stateIndex := v.rp.getScaleSetStateIndex(VMScaleSetName)
	if stateIndex == -1 {
		return fmt.Errorf("ScaleSet %s not found", VMScaleSetName)
	}
	for _, vm := range v.rp.State[stateIndex].Vms {
		if *vm.InstanceID == instanceID {
			v.rp.Log.Debugf("reimaging VM %s:%s", VMScaleSetName, instanceID)
			_, filesExist := v.rp.State[stateIndex].VmsDir[instanceID]
			if filesExist {
				os.RemoveAll(v.rp.State[stateIndex].VmsDir[instanceID])
				delete(v.rp.State[stateIndex].VmsDir, instanceID)
			}
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}

// Restart Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Restart(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Restart:"+VMScaleSetName+":"+instanceID)
	err := v.Deallocate(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return err
	}
	return v.Start(ctx, resourceGroupName, VMScaleSetName, instanceID)
}

// RunCommand Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) RunCommand(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters azcompute.RunCommandInput) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:RunCommand:"+VMScaleSetName+":"+instanceID)
	return nil
}

// Start Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Start(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Start:"+VMScaleSetName+":"+instanceID)
	stateIndex := v.rp.getScaleSetStateIndex(VMScaleSetName)
	if stateIndex == -1 {
		return fmt.Errorf("ScaleSet %s not found", VMScaleSetName)
	}
	for _, vm := range v.rp.State[stateIndex].Vms {
		if *vm.InstanceID == instanceID {
			var err error
			_, filesExist := v.rp.State[stateIndex].VmsDir[instanceID]
			if !filesExist {
				v.rp.State[stateIndex].VmsDir[instanceID], err = ioutil.TempDir("", "fake-"+*vm.OsProfile.ComputerName)
				if err != nil {
					return err
				}
				start, err := startup.New(v.rp.Log, v.rp.Cs, api.TestConfig{})
				if err != nil {
					return err
				}
				err = start.WriteFilesExt(ctx, v.kvc, *vm.OsProfile.ComputerName, "fake-domainname", v.rp.State[stateIndex].VmsDir[instanceID])
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}
