package compute

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
)

type FakeVirtualMachineScaleSetsClient struct {
	rp *ComputeRP
}

// NewFakeVirtualMachineScaleSetsClient creates a new Fake instance
func NewFakeVirtualMachineScaleSetsClient(rp *ComputeRP) *FakeVirtualMachineScaleSetsClient {
	return &FakeVirtualMachineScaleSetsClient{rp: rp}
}

func (s *FakeVirtualMachineScaleSetsClient) scale(ctx context.Context, resourceGroupName string, ss *azcompute.VirtualMachineScaleSet) error {
	_, rgExist := s.rp.Vms[*ss.Name]
	if !rgExist {
		s.rp.Vms[*ss.Name] = []azcompute.VirtualMachineScaleSetVM{}
	}

	have := int64(len(s.rp.Vms[*ss.Name]))
	s.rp.Log.Debugf("scale have:%d, cap:%d", have, *ss.Sku.Capacity)
	if have > *ss.Sku.Capacity {
		s.rp.Vms[*ss.Name] = s.rp.Vms[*ss.Name][:*ss.Sku.Capacity]
		return nil
	}
	for v := have; *ss.Sku.Capacity > have; v++ {
		name := fmt.Sprintf("%s-%d", (*ss.Name)[3:], v)
		compName := fmt.Sprintf("%s-%06d", (*ss.Name)[3:], v)
		s.rp.Log.Infof("scale have:%d, cap:%d, v:%d, name:%s, compName:%s", have, *ss.Sku.Capacity, v, name, compName)
		vm := compute.VirtualMachineScaleSetVM{
			ID:         to.StringPtr(uuid.NewV4().String()),
			InstanceID: to.StringPtr(fmt.Sprintf("%d", v)),
			Name:       &name,
			VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
				OsProfile: &compute.OSProfile{ComputerName: &compName},
			},
			Location: ss.Location,
			Plan:     ss.Plan,
			Tags:     ss.Tags,
			Zones:    ss.Zones,
		}
		s.rp.Vms[*ss.Name] = append(s.rp.Vms[*ss.Name], vm)
		have++
	}
	return nil
}

// CreateOrUpdate Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) CreateOrUpdate(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSet) error {
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
func (s *FakeVirtualMachineScaleSetsClient) Get(ctx context.Context, resourceGroup, VMScaleSetName string) (compute.VirtualMachineScaleSet, error) {
	for _, ss := range s.rp.Ssc {
		if VMScaleSetName == *ss.Name {
			return ss, nil
		}
	}
	return compute.VirtualMachineScaleSet{}, nil
}

// List Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) List(ctx context.Context, resourceGroup string) ([]compute.VirtualMachineScaleSet, error) {
	s.rp.Calls = append(s.rp.Calls, "VirtualMachineScaleSetsClient:List")
	return s.rp.Ssc, nil
}

// Update Fakes base method
func (s *FakeVirtualMachineScaleSetsClient) Update(ctx context.Context, resourceGroupName, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) error {
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
func (s *FakeVirtualMachineScaleSetsClient) UpdateInstances(ctx context.Context, resourceGroupName, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) error {
	s.rp.Calls = append(s.rp.Calls, "VirtualMachineScaleSetsClient:UpdateInstances:"+VMScaleSetName)
	return nil
}
