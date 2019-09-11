package arm

import (
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
)

// VirtualNetwork is an alias for a type from the Azure SDK with custom marshaling
type VirtualNetwork network.VirtualNetwork

// MarshalJSON is the custom marshaler for VirtualNetwork.
func (vn VirtualNetwork) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if vn.VirtualNetworkPropertiesFormat != nil {
		objectMap["properties"] = vn.VirtualNetworkPropertiesFormat
	}
	if vn.Etag != nil {
		objectMap["etag"] = vn.Etag
	}
	if vn.ID != nil {
		objectMap["id"] = vn.ID
	}
	if vn.Name != nil {
		objectMap["name"] = vn.Name
	}
	if vn.Type != nil {
		objectMap["type"] = vn.Type
	}
	if vn.Location != nil {
		objectMap["location"] = vn.Location
	}
	if vn.Tags != nil {
		objectMap["tags"] = vn.Tags
	}
	return json.Marshal(objectMap)
}

// PublicIPAddress is an alias for a type from the Azure SDK with custom marshaling
type PublicIPAddress network.PublicIPAddress

// MarshalJSON is the custom marshaler for PublicIPAddress.
func (pia PublicIPAddress) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if pia.Sku != nil {
		objectMap["sku"] = pia.Sku
	}
	if pia.PublicIPAddressPropertiesFormat != nil {
		objectMap["properties"] = pia.PublicIPAddressPropertiesFormat
	}
	if pia.Etag != nil {
		objectMap["etag"] = pia.Etag
	}
	if pia.Zones != nil {
		objectMap["zones"] = pia.Zones
	}
	if pia.ID != nil {
		objectMap["id"] = pia.ID
	}
	if pia.Name != nil {
		objectMap["name"] = pia.Name
	}
	if pia.Type != nil {
		objectMap["type"] = pia.Type
	}
	if pia.Location != nil {
		objectMap["location"] = pia.Location
	}
	if pia.Tags != nil {
		objectMap["tags"] = pia.Tags
	}
	return json.Marshal(objectMap)
}

// LoadBalancer is an alias for a type from the Azure SDK with custom marshaling
type LoadBalancer network.LoadBalancer

// MarshalJSON is the custom marshaler for LoadBalancer.
func (lb LoadBalancer) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if lb.Sku != nil {
		objectMap["sku"] = lb.Sku
	}
	if lb.LoadBalancerPropertiesFormat != nil {
		objectMap["properties"] = lb.LoadBalancerPropertiesFormat
	}
	if lb.Etag != nil {
		objectMap["etag"] = lb.Etag
	}
	if lb.ID != nil {
		objectMap["id"] = lb.ID
	}
	if lb.Name != nil {
		objectMap["name"] = lb.Name
	}
	if lb.Type != nil {
		objectMap["type"] = lb.Type
	}
	if lb.Location != nil {
		objectMap["location"] = lb.Location
	}
	if lb.Tags != nil {
		objectMap["tags"] = lb.Tags
	}
	return json.Marshal(objectMap)
}

// Account is an alias for a type from the Azure SDK with custom marshaling
type Account storage.Account

// MarshalJSON is the custom marshaler for Account.
func (a Account) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if a.Sku != nil {
		objectMap["sku"] = a.Sku
	}
	if a.Kind != "" {
		objectMap["kind"] = a.Kind
	}
	if a.Identity != nil {
		objectMap["identity"] = a.Identity
	}
	if a.AccountProperties != nil {
		objectMap["properties"] = a.AccountProperties
	}
	if a.Tags != nil {
		objectMap["tags"] = a.Tags
	}
	if a.Location != nil {
		objectMap["location"] = a.Location
	}
	if a.ID != nil {
		objectMap["id"] = a.ID
	}
	if a.Name != nil {
		objectMap["name"] = a.Name
	}
	if a.Type != nil {
		objectMap["type"] = a.Type
	}
	return json.Marshal(objectMap)
}

// SecurityGroup is an alias for a type from the Azure SDK with custom marshaling
type SecurityGroup network.SecurityGroup

// MarshalJSON is the custom marshaler for SecurityGroup.
func (sg SecurityGroup) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if sg.SecurityGroupPropertiesFormat != nil {
		objectMap["properties"] = sg.SecurityGroupPropertiesFormat
	}
	if sg.Etag != nil {
		objectMap["etag"] = sg.Etag
	}
	if sg.ID != nil {
		objectMap["id"] = sg.ID
	}
	if sg.Name != nil {
		objectMap["name"] = sg.Name
	}
	if sg.Type != nil {
		objectMap["type"] = sg.Type
	}
	if sg.Location != nil {
		objectMap["location"] = sg.Location
	}
	if sg.Tags != nil {
		objectMap["tags"] = sg.Tags
	}
	return json.Marshal(objectMap)
}

// VirtualMachineScaleSet is an alias for a type from the Azure SDK with custom marshaling
type VirtualMachineScaleSet compute.VirtualMachineScaleSet

// MarshalJSON is the custom marshaler for VirtualMachineScaleSet.
func (vmss VirtualMachineScaleSet) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if vmss.Sku != nil {
		objectMap["sku"] = vmss.Sku
	}
	if vmss.Plan != nil {
		objectMap["plan"] = vmss.Plan
	}
	if vmss.VirtualMachineScaleSetProperties != nil {
		objectMap["properties"] = vmss.VirtualMachineScaleSetProperties
	}
	if vmss.Identity != nil {
		objectMap["identity"] = vmss.Identity
	}
	if vmss.Zones != nil {
		objectMap["zones"] = vmss.Zones
	}
	if vmss.ID != nil {
		objectMap["id"] = vmss.ID
	}
	if vmss.Name != nil {
		objectMap["name"] = vmss.Name
	}
	if vmss.Type != nil {
		objectMap["type"] = vmss.Type
	}
	if vmss.Location != nil {
		objectMap["location"] = vmss.Location
	}
	if vmss.Tags != nil {
		objectMap["tags"] = vmss.Tags
	}
	return json.Marshal(objectMap)
}

// VirtualMachine is an alias for a type from the Azure SDK with custom marshaling
type VirtualMachine compute.VirtualMachine

// MarshalJSON is the custom marshaler for VirtualMachine.
func (VM VirtualMachine) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if VM.Plan != nil {
		objectMap["plan"] = VM.Plan
	}
	if VM.VirtualMachineProperties != nil {
		objectMap["properties"] = VM.VirtualMachineProperties
	}
	if VM.Resources != nil {
		objectMap["resources"] = VM.Resources
	}
	if VM.Identity != nil {
		objectMap["identity"] = VM.Identity
	}
	if VM.Zones != nil {
		objectMap["zones"] = VM.Zones
	}
	if VM.ID != nil {
		objectMap["id"] = VM.ID
	}
	if VM.Name != nil {
		objectMap["name"] = VM.Name
	}
	if VM.Type != nil {
		objectMap["type"] = VM.Type
	}
	if VM.Location != nil {
		objectMap["location"] = VM.Location
	}
	if VM.Tags != nil {
		objectMap["tags"] = VM.Tags
	}
	return json.Marshal(objectMap)
}

// VirtualMachineExtension is an alias for a type from the Azure SDK with custom marshaling
type VirtualMachineExtension compute.VirtualMachineExtension

// MarshalJSON is the custom marshaler for VirtualMachineExtension.
func (vme VirtualMachineExtension) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if vme.VirtualMachineExtensionProperties != nil {
		objectMap["properties"] = vme.VirtualMachineExtensionProperties
	}
	if vme.ID != nil {
		objectMap["id"] = vme.ID
	}
	if vme.Name != nil {
		objectMap["name"] = vme.Name
	}
	if vme.Type != nil {
		objectMap["type"] = vme.Type
	}
	if vme.Location != nil {
		objectMap["location"] = vme.Location
	}
	if vme.Tags != nil {
		objectMap["tags"] = vme.Tags
	}
	return json.Marshal(objectMap)
}

// Interface is an alias for a type from the Azure SDK with custom marshaling
type Interface network.Interface

// MarshalJSON is the custom marshaler for Interface.
func (i Interface) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if i.InterfacePropertiesFormat != nil {
		objectMap["properties"] = i.InterfacePropertiesFormat
	}
	if i.Etag != nil {
		objectMap["etag"] = i.Etag
	}
	if i.ID != nil {
		objectMap["id"] = i.ID
	}
	if i.Name != nil {
		objectMap["name"] = i.Name
	}
	if i.Type != nil {
		objectMap["type"] = i.Type
	}
	if i.Location != nil {
		objectMap["location"] = i.Location
	}
	if i.Tags != nil {
		objectMap["tags"] = i.Tags
	}
	return json.Marshal(objectMap)
}
