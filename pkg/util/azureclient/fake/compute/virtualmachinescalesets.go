package compute

import (
	"context"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/compute"
)

type FakeVirtualMachineScaleSetsClient struct {
	rp  *ComputeRP
	vms compute.VirtualMachineScaleSetVMsClient
}

var _ compute.VirtualMachineScaleSetsClient = &FakeVirtualMachineScaleSetsClient{}

// NewFakeVirtualMachineScaleSetsClient creates a new Fake instance
func NewFakeVirtualMachineScaleSetsClient(vms compute.VirtualMachineScaleSetVMsClient, rp *ComputeRP) *FakeVirtualMachineScaleSetsClient {
	return &FakeVirtualMachineScaleSetsClient{rp: rp, vms: vms}
}

func (s *FakeVirtualMachineScaleSetsClient) scale(ctx context.Context, resourceGroupName string, ss *azcompute.VirtualMachineScaleSet) error {
	stateIndex := s.rp.getScaleSetStateIndex(*ss.Name)
	have := int64(len(s.rp.State[stateIndex].Vms))
	if have > *ss.Sku.Capacity {
		for ix := range s.rp.State[stateIndex].Vms[*ss.Sku.Capacity:have] {
			err := s.vms.Delete(ctx, resourceGroupName, *ss.Name, *s.rp.State[stateIndex].Vms[ix].InstanceID)
			if err != nil {
				return err
			}
		}
		return nil
	}
	for v := have; *ss.Sku.Capacity > have; v++ {
		err := s.vms.Start(ctx, resourceGroupName, *ss.Name, s.rp.createVM(ss, stateIndex, int(v)))
		if err != nil {
			return err
		}
		have++
	}
	return nil
}

// CreateOrUpdate Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) CreateOrUpdate(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters azcompute.VirtualMachineScaleSet) error {
	s.rp.Calls = append(s.rp.Calls, "VirtualMachineScaleSetsClient:CreateOrUpdate:"+VMScaleSetName)
	found := false
	for i, ss := range s.rp.Ssc {
		if VMScaleSetName == *ss.Name {
			found = true
			s.rp.Ssc[i] = parameters
			break
		}
	}
	if !found {
		s.rp.Ssc = append(s.rp.Ssc, parameters)
	}

	if s.rp.getScaleSetStateIndex(VMScaleSetName) == -1 {
		s.rp.State = append(s.rp.State, ScaleSetState{
			Name:   VMScaleSetName,
			Vms:    []azcompute.VirtualMachineScaleSetVM{},
			VmsDir: map[string]string{},
		})
	}

	return s.scale(ctx, resourceGroupName, &parameters)
}

// Delete Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) Delete(ctx context.Context, resourceGroupName, VMScaleSetName string) error {
	s.rp.Calls = append(s.rp.Calls, "VirtualMachineScaleSetsClient:Delete:"+VMScaleSetName)
	for i, ss := range s.rp.Ssc {
		if VMScaleSetName == *ss.Name {
			*ss.Sku.Capacity = 0
			err := s.scale(ctx, resourceGroupName, &ss)
			if err != nil {
				return err
			}
			s.rp.Ssc = append(s.rp.Ssc[:i], s.rp.Ssc[i+1:]...)
			return nil
		}
	}
	return nil
}

// Get Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) Get(ctx context.Context, resourceGroup, VMScaleSetName string) (azcompute.VirtualMachineScaleSet, error) {
	for _, ss := range s.rp.Ssc {
		if VMScaleSetName == *ss.Name {
			return ss, nil
		}
	}
	return azcompute.VirtualMachineScaleSet{}, nil
}

// List Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) List(ctx context.Context, resourceGroup string) ([]azcompute.VirtualMachineScaleSet, error) {
	s.rp.Calls = append(s.rp.Calls, "VirtualMachineScaleSetsClient:List")
	return s.rp.Ssc, nil
}

// Update Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) Update(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters azcompute.VirtualMachineScaleSetUpdate) error {
	s.rp.Calls = append(s.rp.Calls, "VirtualMachineScaleSetsClient:Update:"+VMScaleSetName)
	for _, ss := range s.rp.Ssc {
		if VMScaleSetName == *ss.Name {
			// the scaler changes the capacity
			ss.Sku.Capacity = parameters.Sku.Capacity
			return s.scale(ctx, resourceGroupName, &ss)
		}
	}
	return nil
}

// UpdateInstances Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) UpdateInstances(ctx context.Context, resourceGroupName, VMScaleSetName string, VMInstanceIDs azcompute.VirtualMachineScaleSetVMInstanceRequiredIDs) error {
	s.rp.Calls = append(s.rp.Calls, "VirtualMachineScaleSetsClient:UpdateInstances:"+VMScaleSetName)
	return nil
}
