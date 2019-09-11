package compute

import (
	"context"
	"fmt"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
)

type ComputeRP struct {
	Log   *logrus.Entry
	Calls []string
	Vms   map[string][]azcompute.VirtualMachineScaleSetVM
	Ssc   []azcompute.VirtualMachineScaleSet
}

type FakeVirtualMachineScaleSetVMsClient struct {
	rp *ComputeRP
}

// NewFakeVirtualMachineScaleSetVMsClient creates a new Fake instance
func NewFakeVirtualMachineScaleSetVMsClient(rp *ComputeRP) *FakeVirtualMachineScaleSetVMsClient {
	return &FakeVirtualMachineScaleSetVMsClient{rp: rp}
}

// Deallocate Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Deallocate:"+VMScaleSetName+":"+instanceID)
	for _, vm := range v.rp.Vms[VMScaleSetName] {
		if *vm.InstanceID == instanceID {
			vm.VirtualMachineScaleSetVMProperties.ProvisioningState = to.StringPtr("Stopped")
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}

// Delete Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Delete:"+VMScaleSetName+":"+instanceID)
	for s, vm := range v.rp.Vms[VMScaleSetName] {
		if *vm.InstanceID == instanceID {
			v.rp.Vms[VMScaleSetName] = append(v.rp.Vms[VMScaleSetName][:s], v.rp.Vms[VMScaleSetName][s+1:]...)
			return nil
		}
	}
	return nil
}

// List Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName, VMScaleSetName, filter, selectParameter, expand string) ([]azcompute.VirtualMachineScaleSetVM, error) {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:List:"+VMScaleSetName)
	return v.rp.Vms[VMScaleSetName], nil
}

// Reimage Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Reimage(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string, VMScaleSetVMReimageInput *azcompute.VirtualMachineScaleSetVMReimageParameters) error {
	v.rp.Calls = append(v.rp.Calls, "VirtualMachineScaleSetVMsClient:Reimage:"+VMScaleSetName+":"+instanceID)
	for _, vm := range v.rp.Vms[VMScaleSetName] {
		if *vm.InstanceID == instanceID {
			vm.VirtualMachineScaleSetVMProperties.ProvisioningState = to.StringPtr("Reimaged")
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
	for _, vm := range v.rp.Vms[VMScaleSetName] {
		if *vm.InstanceID == instanceID {
			vm.VirtualMachineScaleSetVMProperties.ProvisioningState = to.StringPtr("Started")
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}
