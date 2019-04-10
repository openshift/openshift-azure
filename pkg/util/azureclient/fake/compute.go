package fake

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
)

type FakeVirtualMachineScaleSetVMsClient struct {
	az *AzureCloud
}

// NewFakeVirtualMachineScaleSetVMsClient creates a new Fake instance
func NewFakeVirtualMachineScaleSetVMsClient(az *AzureCloud) *FakeVirtualMachineScaleSetVMsClient {
	return &FakeVirtualMachineScaleSetVMsClient{az: az}
}

// Client Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Client() autorest.Client {
	return allwaysDoneClient()
}

// Deallocate Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) error {
	for _, vm := range v.az.Vms {
		if *vm.InstanceID == instanceID {
			vm.VirtualMachineScaleSetVMProperties.ProvisioningState = to.StringPtr("Stopped")
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}

// Delete Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) error {
	for s, vm := range v.az.Vms {
		if *vm.InstanceID == instanceID {
			v.az.Vms = append(v.az.Vms[:s], v.az.Vms[s+1:]...)
			return nil
		}
	}
	return nil
}

// List Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) List(ctx context.Context, resourceGroupName, virtualMachineScaleSetName, filter, selectParameter, expand string) ([]compute.VirtualMachineScaleSetVM, error) {
	prefix := virtualMachineScaleSetName[3:]
	result := []compute.VirtualMachineScaleSetVM{}
	for _, vm := range v.az.Vms {
		if strings.HasPrefix(*vm.Name, prefix) {
			result = append(result, vm)
		}
	}

	return result, nil
}

// Reimage Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Reimage(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string, VMScaleSetVMReimageInput *compute.VirtualMachineScaleSetVMReimageParameters) error {
	for _, vm := range v.az.Vms {
		if *vm.InstanceID == instanceID {
			vm.VirtualMachineScaleSetVMProperties.ProvisioningState = to.StringPtr("Reimaged")
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}

// Restart Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Restart(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	err := v.Deallocate(ctx, resourceGroupName, VMScaleSetName, instanceID)
	if err != nil {
		return err
	}
	return v.Start(ctx, resourceGroupName, VMScaleSetName, instanceID)
}

// RunCommand Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) RunCommand(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string, parameters compute.RunCommandInput) error {
	return nil
}

// Start Fakes base method
func (v *FakeVirtualMachineScaleSetVMsClient) Start(ctx context.Context, resourceGroupName, VMScaleSetName, instanceID string) error {
	for _, vm := range v.az.Vms {
		if *vm.InstanceID == instanceID {
			vm.VirtualMachineScaleSetVMProperties.ProvisioningState = to.StringPtr("Started")
			return nil
		}
	}
	return fmt.Errorf("VM %s/%s not found", VMScaleSetName, instanceID)
}

////////////////////////////////////////////////////////////////////////////////

type FakeVirtualMachineScaleSetsClient struct {
	az *AzureCloud
}

// NewFakeVirtualMachineScaleSetsClient creates a new Fake instance
func NewFakeVirtualMachineScaleSetsClient(az *AzureCloud) *FakeVirtualMachineScaleSetsClient {
	return &FakeVirtualMachineScaleSetsClient{az: az}
}

// Client Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) Client() autorest.Client {
	return allwaysDoneClient()
}

func (s *FakeVirtualMachineScaleSetsClient) scale(ctx context.Context, resourceGroupName string, ss *compute.VirtualMachineScaleSet) error {
	var have int64
	for _, vm := range s.az.Vms {
		if *ss.Name == *vm.Tags["scaleset"] {
			have++
		}
	}
	s.az.log.Debugf("scale have:%d, cap:%d", have, *ss.Sku.Capacity)
	if have > *ss.Sku.Capacity {
		return fmt.Errorf("should not be automatically scaling down")
	}
	for v := have; *ss.Sku.Capacity > have; v++ {
		name := fmt.Sprintf("%s-%d", (*ss.Name)[3:], v)
		compName := fmt.Sprintf("%s-%06d", (*ss.Name)[3:], v)
		s.az.log.Infof("scale have:%d, cap:%d, v:%d, name:%s, compName:%s", have, *ss.Sku.Capacity, v, name, compName)
		tags := ss.Tags
		if tags == nil {
			tags = map[string]*string{}
		}
		tags["scaleset"] = ss.Name
		vm := compute.VirtualMachineScaleSetVM{
			ID:         to.StringPtr(uuid.NewV4().String()),
			InstanceID: to.StringPtr(uuid.NewV4().String()),
			Name:       &name,
			VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
				OsProfile: &compute.OSProfile{ComputerName: &compName},
			},
			Location: ss.Location,
			Plan:     ss.Plan,
			Tags:     tags,
			Zones:    ss.Zones,
		}
		s.az.Vms = append(s.az.Vms, vm)
		s.az.VirtualMachineScaleSetVMsClient.Start(ctx, resourceGroupName, *ss.Name, *vm.InstanceID)
		have++
	}
	return nil
}

// CreateOrUpdate Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) CreateOrUpdate(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSet) error {
	found := false
	for i, ss := range s.az.Ssc {
		if VMScaleSetName == *ss.Name {
			found = true
			s.az.Ssc[i] = parameters
			break
		}
	}
	if !found {
		s.az.Ssc = append(s.az.Ssc, parameters)
	}
	s.scale(ctx, resourceGroupName, &parameters)
	return nil
}

// Delete Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) Delete(ctx context.Context, resourceGroupName, VMScaleSetName string) error {
	for i, ss := range s.az.Ssc {
		if VMScaleSetName == *ss.Name {
			*ss.Sku.Capacity = 0
			s.scale(ctx, resourceGroupName, &ss)
			s.az.Ssc = append(s.az.Ssc[:i], s.az.Ssc[i+1:]...)
			return nil
		}
	}
	return nil
}

// List Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) List(ctx context.Context, resourceGroup string) ([]compute.VirtualMachineScaleSet, error) {
	return s.az.Ssc, nil
}

// Update Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) Update(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) error {
	for _, ss := range s.az.Ssc {
		if VMScaleSetName == *ss.Name {
			// the scaler changes the capacity
			ss.Sku.Capacity = parameters.Sku.Capacity
			s.scale(ctx, resourceGroupName, &ss)
			return nil
		}
	}
	return nil
}

// UpdateInstances Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) UpdateInstances(ctx context.Context, resourceGroupName, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) error {
	// not implemented yet
	return nil
}
