package fakerp

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/network"
)

const (
	// DefaultBaseURI is the default URI used for the service Network
	DefaultBaseURI = "https://management.azure.com"
)

type networkManager struct {
	pec                     network.PrivateEndpointsClient
	plsc                    network.PrivateLinkServicesClient
	nic                     network.InterfacesClient
	managementResourceGroup string
	log                     *logrus.Entry
}

func newNetworkManager(ctx context.Context, log *logrus.Entry, subscriptionID, resourceGroupName string) (*networkManager, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	return &networkManager{
		pec:                     network.NewPrivateEndpointsClient(ctx, log, subscriptionID, authorizer),
		nic:                     network.NewInterfacesClient(ctx, log, subscriptionID, authorizer),
		plsc:                    network.NewPrivateLinkServicesClient(ctx, log, subscriptionID, authorizer),
		managementResourceGroup: resourceGroupName,
		log:                     log,
	}, nil
}

func (nm *networkManager) deletePLSPE(ctx context.Context, resourceGroupname, resourceName string) error {
	pls, err := nm.plsc.Get(ctx, resourceGroupname, resourceName, "")
	if err != nil {
		return err
	}
	for _, pe := range *pls.PrivateEndpointConnections {
		resources, err := azure.ParseResourceID(*pe.PrivateEndpointConnectionProperties.PrivateEndpoint.ID)
		if err != nil {
			return err
		}

		_, err = nm.pec.Delete(ctx, nm.managementResourceGroup, resources.ResourceName)
		if err != nil {
			return err
		}
	}

	return err
}

func (nm *networkManager) getPrivateEndpointIP(ctx context.Context, resourceName string) (*string, error) {
	pe, err := nm.pec.Get(ctx, nm.managementResourceGroup, resourceName, "")
	if err != nil {
		return nil, err
	}

	for _, nic := range *pe.NetworkInterfaces {
		resource, err := azure.ParseResourceID(*nic.ID)
		if err != nil {
			return nil, err
		}
		ni, err := nm.nic.Get(ctx, resource.ResourceGroup, resource.ResourceName, "")
		if err != nil {
			return nil, err
		}
		for _, ip := range *ni.IPConfigurations {
			if ip.PrivateIPAddress != nil {
				return ip.PrivateIPAddress, nil
			}
		}
	}

	return nil, err
}
