package privatelink

// This file is a copy from SDK 32+ PrivateLink components to enable usage of PLS and SDK v24
// FIXME: This should be removed when revendoring SDK v32+

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

// PrivateLinkService private link service resource.
type PrivateLinkService struct {
	autorest.Response `json:"-"`
	// PrivateLinkServiceProperties - Properties of the private link service.
	*PrivateLinkServiceProperties `json:"properties,omitempty"`
	// Etag - Gets a unique read-only string that changes whenever the resource is updated.
	Etag *string `json:"etag,omitempty"`
	// ID - Resource ID.
	ID *string `json:"id,omitempty"`
	// Name - READ-ONLY; Resource name.
	Name *string `json:"name,omitempty"`
	// Type - READ-ONLY; Resource type.
	Type *string `json:"type,omitempty"`
	// Version - Resource version.
	Version *string `json:"apiVersion,omitempty"`
	// Location - Resource location.
	Location *string `json:"location,omitempty"`
	// Tags - Resource tags.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON is the custom marshaler for PrivateLinkService.
func (pls PrivateLinkService) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if pls.PrivateLinkServiceProperties != nil {
		objectMap["properties"] = pls.PrivateLinkServiceProperties
	}
	if pls.Etag != nil {
		objectMap["etag"] = pls.Etag
	}
	if pls.ID != nil {
		objectMap["id"] = pls.ID
	}
	if pls.Location != nil {
		objectMap["location"] = pls.Location
	}
	if pls.Tags != nil {
		objectMap["tags"] = pls.Tags
	}
	if pls.Type != nil {
		objectMap["type"] = pls.Type
	}
	if pls.Version != nil {
		objectMap["apiVersion"] = pls.Version
	}
	if pls.Name != nil {
		objectMap["name"] = pls.Name
	}
	return json.Marshal(objectMap)
}

// UnmarshalJSON is the custom unmarshaler for PrivateLinkService struct.
func (pls *PrivateLinkService) UnmarshalJSON(body []byte) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(body, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		switch k {
		case "properties":
			if v != nil {
				var privateLinkServiceProperties PrivateLinkServiceProperties
				err = json.Unmarshal(*v, &privateLinkServiceProperties)
				if err != nil {
					return err
				}
				pls.PrivateLinkServiceProperties = &privateLinkServiceProperties
			}
		case "etag":
			if v != nil {
				var etag string
				err = json.Unmarshal(*v, &etag)
				if err != nil {
					return err
				}
				pls.Etag = &etag
			}
		case "id":
			if v != nil {
				var ID string
				err = json.Unmarshal(*v, &ID)
				if err != nil {
					return err
				}
				pls.ID = &ID
			}
		case "name":
			if v != nil {
				var name string
				err = json.Unmarshal(*v, &name)
				if err != nil {
					return err
				}
				pls.Name = &name
			}
		case "type":
			if v != nil {
				var typeVar string
				err = json.Unmarshal(*v, &typeVar)
				if err != nil {
					return err
				}
				pls.Type = &typeVar
			}
		case "location":
			if v != nil {
				var location string
				err = json.Unmarshal(*v, &location)
				if err != nil {
					return err
				}
				pls.Location = &location
			}
		case "apiVersion":
			if v != nil {
				var version string
				err = json.Unmarshal(*v, &version)
				if err != nil {
					return err
				}
				pls.Version = &version
			}
		case "tags":
			if v != nil {
				var tags map[string]*string
				err = json.Unmarshal(*v, &tags)
				if err != nil {
					return err
				}
				pls.Tags = tags
			}

		}
	}

	return nil
}

// PrivateLinkServiceProperties properties of the private link service.
type PrivateLinkServiceProperties struct {
	// LoadBalancerFrontendIPConfigurations - An array of references to the load balancer IP configurations.
	LoadBalancerFrontendIPConfigurations *[]network.FrontendIPConfiguration `json:"loadBalancerFrontendIpConfigurations,omitempty"`
	// IPConfigurations - An array of references to the private link service IP configuration.
	IPConfigurations *[]PrivateLinkServiceIPConfiguration `json:"ipConfigurations,omitempty"`
	// NetworkInterfaces - READ-ONLY; Gets an array of references to the network interfaces created for this private link service.
	NetworkInterfaces *[]network.Interface `json:"networkInterfaces,omitempty"`
	// ProvisioningState - READ-ONLY; The provisioning state of the private link service. Possible values are: 'Updating', 'Succeeded', and 'Failed'.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// PrivateEndpointConnections - An array of list about connections to the private endpoint.
	PrivateEndpointConnections *[]PrivateEndpointConnection `json:"privateEndpointConnections,omitempty"`
	// Visibility - The visibility list of the private link service.
	Visibility *PrivateLinkServicePropertiesVisibility `json:"visibility,omitempty"`
	// AutoApproval - The auto-approval list of the private link service.
	AutoApproval *PrivateLinkServicePropertiesAutoApproval `json:"autoApproval,omitempty"`
	// Fqdns - The list of Fqdn.
	Fqdns *[]string `json:"fqdns,omitempty"`
	// Alias - READ-ONLY; The alias of the private link service.
	Alias *string `json:"alias,omitempty"`
}

// PrivateLinkServiceIPConfiguration the private link service ip configuration.
type PrivateLinkServiceIPConfiguration struct {
	// PrivateLinkServiceIPConfigurationProperties - Properties of the private link service ip configuration.
	*PrivateLinkServiceIPConfigurationProperties `json:"properties,omitempty"`
	// Name - The name of private link service ip configuration.
	Name *string `json:"name,omitempty"`
}

// PrivateLinkServiceIPConfigurationProperties properties of private link service IP configuration.
type PrivateLinkServiceIPConfigurationProperties struct {
	// PrivateIPAddress - The private IP address of the IP configuration.
	PrivateIPAddress *string `json:"privateIPAddress,omitempty"`
	// PrivateIPAllocationMethod - The private IP address allocation method. Possible values include: 'Static', 'Dynamic'
	PrivateIPAllocationMethod network.IPAllocationMethod `json:"privateIPAllocationMethod,omitempty"`
	// Subnet - The reference of the subnet resource.
	Subnet *network.Subnet `json:"subnet,omitempty"`
	// PublicIPAddress - The reference of the public IP resource.
	PublicIPAddress *network.PublicIPAddress `json:"publicIPAddress,omitempty"`
	// ProvisioningState - Gets the provisioning state of the public IP resource. Possible values are: 'Updating', 'Deleting', and 'Failed'.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// PrivateIPAddressVersion - Available from Api-Version 2016-03-30 onwards, it represents whether the specific ipconfiguration is IPv4 or IPv6. Default is taken as IPv4. Possible values include: 'IPv4', 'IPv6'
	PrivateIPAddressVersion network.IPVersion `json:"privateIPAddressVersion,omitempty"`
}

// PrivateEndpointConnection privateEndpointConnection resource.
type PrivateEndpointConnection struct {
	autorest.Response `json:"-"`
	// PrivateEndpointConnectionProperties - Properties of the private end point connection.
	*PrivateEndpointConnectionProperties `json:"properties,omitempty"`
	// Name - The name of the resource that is unique within a resource group. This name can be used to access the resource.
	Name *string `json:"name,omitempty"`
	// ID - Resource ID.
	ID *string `json:"id,omitempty"`
}

// PrivateEndpointConnectionProperties properties of the PrivateEndpointConnectProperties.
type PrivateEndpointConnectionProperties struct {
	// PrivateEndpoint - The resource of private end point.
	PrivateEndpoint *PrivateEndpoint `json:"privateEndpoint,omitempty"`
	// PrivateLinkServiceConnectionState - A collection of information about the state of the connection between service consumer and provider.
	PrivateLinkServiceConnectionState *PrivateLinkServiceConnectionState `json:"privateLinkServiceConnectionState,omitempty"`
}

// PrivateEndpoint private endpoint resource.
type PrivateEndpoint struct {
	autorest.Response `json:"-"`
	// PrivateEndpointProperties - Properties of the private endpoint.
	*PrivateEndpointProperties `json:"properties,omitempty"`
	// Etag - Gets a unique read-only string that changes whenever the resource is updated.
	Etag *string `json:"etag,omitempty"`
	// ID - Resource ID.
	ID *string `json:"id,omitempty"`
	// Name - READ-ONLY; Resource name.
	Name *string `json:"name,omitempty"`
	// Type - READ-ONLY; Resource type.
	Type *string `json:"type,omitempty"`
	// Version - Resource location.
	Version *string `json:"apiVersion,omitempty"`
	// Location - Resource location.
	Location *string `json:"location,omitempty"`
	// Tags - Resource tags.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON is the custom marshaler for PrivateEndpoint.
func (peVar PrivateEndpoint) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if peVar.PrivateEndpointProperties != nil {
		objectMap["properties"] = peVar.PrivateEndpointProperties
	}
	if peVar.Etag != nil {
		objectMap["etag"] = peVar.Etag
	}
	if peVar.ID != nil {
		objectMap["id"] = peVar.ID
	}
	if peVar.Location != nil {
		objectMap["location"] = peVar.Location
	}
	if peVar.Type != nil {
		objectMap["type"] = peVar.Type
	}
	if peVar.Version != nil {
		objectMap["apiVersion"] = peVar.Version
	}
	if peVar.Name != nil {
		objectMap["name"] = peVar.Name
	}
	if peVar.Tags != nil {
		objectMap["tags"] = peVar.Tags
	}
	return json.Marshal(objectMap)
}

// UnmarshalJSON is the custom unmarshaler for PrivateEndpoint struct.
func (peVar *PrivateEndpoint) UnmarshalJSON(body []byte) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(body, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		switch k {
		case "properties":
			if v != nil {
				var privateEndpointProperties PrivateEndpointProperties
				err = json.Unmarshal(*v, &privateEndpointProperties)
				if err != nil {
					return err
				}
				peVar.PrivateEndpointProperties = &privateEndpointProperties
			}
		case "etag":
			if v != nil {
				var etag string
				err = json.Unmarshal(*v, &etag)
				if err != nil {
					return err
				}
				peVar.Etag = &etag
			}
		case "id":
			if v != nil {
				var ID string
				err = json.Unmarshal(*v, &ID)
				if err != nil {
					return err
				}
				peVar.ID = &ID
			}
		case "name":
			if v != nil {
				var name string
				err = json.Unmarshal(*v, &name)
				if err != nil {
					return err
				}
				peVar.Name = &name
			}
		case "type":
			if v != nil {
				var typeVar string
				err = json.Unmarshal(*v, &typeVar)
				if err != nil {
					return err
				}
				peVar.Type = &typeVar
			}
		case "location":
			if v != nil {
				var location string
				err = json.Unmarshal(*v, &location)
				if err != nil {
					return err
				}
				peVar.Location = &location
			}
		case "apiVersion":
			if v != nil {
				var verion string
				err = json.Unmarshal(*v, &verion)
				if err != nil {
					return err
				}
				peVar.Version = &verion
			}
		case "tags":
			if v != nil {
				var tags map[string]*string
				err = json.Unmarshal(*v, &tags)
				if err != nil {
					return err
				}
				peVar.Tags = tags
			}
		}
	}

	return nil
}

// PrivateEndpointProperties properties of the private endpoint.
type PrivateEndpointProperties struct {
	// Subnet - The ID of the subnet from which the private IP will be allocated.
	Subnet *network.Subnet `json:"subnet,omitempty"`
	// NetworkInterfaces - READ-ONLY; Gets an array of references to the network interfaces created for this private endpoint.
	NetworkInterfaces *[]network.Interface `json:"networkInterfaces,omitempty"`
	// ProvisioningState - READ-ONLY; The provisioning state of the private endpoint. Possible values are: 'Updating', 'Deleting', and 'Failed'.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// PrivateLinkServiceConnections - A grouping of information about the connection to the remote resource.
	PrivateLinkServiceConnections *[]PrivateLinkServiceConnection `json:"privateLinkServiceConnections,omitempty"`
	// ManualPrivateLinkServiceConnections - A grouping of information about the connection to the remote resource. Used when the network admin does not have access to approve connections to the remote resource.
	ManualPrivateLinkServiceConnections *[]PrivateLinkServiceConnection `json:"manualPrivateLinkServiceConnections,omitempty"`
}

// PrivateLinkServiceConnection privateLinkServiceConnection resource.
type PrivateLinkServiceConnection struct {
	// PrivateLinkServiceConnectionProperties - Properties of the private link service connection.
	*PrivateLinkServiceConnectionProperties `json:"properties,omitempty"`
	// Name - The name of the resource that is unique within a resource group. This name can be used to access the resource.
	Name *string `json:"name,omitempty"`
	// ID - Resource ID.
	ID *string `json:"id,omitempty"`
}

// PrivateLinkServiceConnectionProperties properties of the PrivateLinkServiceConnection.
type PrivateLinkServiceConnectionProperties struct {
	// PrivateLinkServiceID - The resource id of private link service.
	PrivateLinkServiceID *string `json:"privateLinkServiceId,omitempty"`
	// GroupIds - The ID(s) of the group(s) obtained from the remote resource that this private endpoint should connect to.
	GroupIds *[]string `json:"groupIds,omitempty"`
	// RequestMessage - A message passed to the owner of the remote resource with this connection request. Restricted to 140 chars.
	RequestMessage *string `json:"requestMessage,omitempty"`
	// PrivateLinkServiceConnectionState - A collection of read-only information about the state of the connection to the remote resource.
	PrivateLinkServiceConnectionState *PrivateLinkServiceConnectionState `json:"privateLinkServiceConnectionState,omitempty"`
}

// PrivateLinkServiceConnectionState a collection of information about the state of the connection between
// service consumer and provider.
type PrivateLinkServiceConnectionState struct {
	// Status - Indicates whether the connection has been Approved/Rejected/Removed by the owner of the service.
	Status *string `json:"status,omitempty"`
	// Description - The reason for approval/rejection of the connection.
	Description *string `json:"description,omitempty"`
	// ActionRequired - A message indicating if changes on the service provider require any updates on the consumer.
	ActionRequired *string `json:"actionRequired,omitempty"`
}

// PrivateLinkServicePropertiesVisibility the visibility list of the private link service.
type PrivateLinkServicePropertiesVisibility struct {
	// Subscriptions - The list of subscriptions.
	Subscriptions *[]string `json:"subscriptions,omitempty"`
}

// PrivateLinkServicePropertiesAutoApproval the auto-approval list of the private link service.
type PrivateLinkServicePropertiesAutoApproval struct {
	// Subscriptions - The list of subscriptions.
	Subscriptions *[]string `json:"subscriptions,omitempty"`
}

// PrivateEndpointsCreateOrUpdateFuture an abstraction for monitoring and retrieving the results of a
// long-running operation.
type PrivateEndpointsCreateOrUpdateFuture struct {
	azure.Future
}

// PrivateEndpointsDeleteFuture an abstraction for monitoring and retrieving the results of a long-running
// operation.
type PrivateEndpointsDeleteFuture struct {
	azure.Future
}

// PrivateEndpointListResultPage contains a page of PrivateEndpoint values.
type PrivateEndpointListResultPage struct {
	fn   func(context.Context, PrivateEndpointListResult) (PrivateEndpointListResult, error)
	pelr PrivateEndpointListResult
}

// PrivateEndpointListResult response for the ListPrivateEndpoints API service call.
type PrivateEndpointListResult struct {
	autorest.Response `json:"-"`
	// Value - Gets a list of private endpoint resources in a resource group.
	Value *[]PrivateEndpoint `json:"value,omitempty"`
	// NextLink - READ-ONLY; The URL to get the next set of results.
	NextLink *string `json:"nextLink,omitempty"`
}

// privateEndpointListResultPreparer prepares a request to retrieve the next set of results.
// It returns nil if no more results exist.
func (pelr PrivateEndpointListResult) privateEndpointListResultPreparer(ctx context.Context) (*http.Request, error) {
	if pelr.NextLink == nil || len(to.String(pelr.NextLink)) < 1 {
		return nil, nil
	}
	return autorest.Prepare((&http.Request{}).WithContext(ctx),
		autorest.AsJSON(),
		autorest.AsGet(),
		autorest.WithBaseURL(to.String(pelr.NextLink)))
}
