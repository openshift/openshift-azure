package fakerp

import (
	"context"
	"fmt"

	aznetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	armconst "github.com/openshift/openshift-azure/pkg/fakerp/arm/constants"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/network"
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

// GetPrivateEndpointIP wraps networkManager creation and getPrivateEndpointIP so it could be called in short form.
func GetPrivateEndpointIP(ctx context.Context, log *logrus.Entry, subscriptionID, managementResourceGroupName, resourceGroupName string) (*string, error) {
	nm, err := newNetworkManager(ctx, log, subscriptionID, managementResourceGroupName)
	if err != nil {
		return nil, err
	}
	exist := nm.privateEndpointExists(ctx, fmt.Sprintf("%s-%s", armconst.PrivateEndpointNamePrefix, resourceGroupName))
	if exist {
		peIP, err := nm.getPrivateEndpointIP(ctx, fmt.Sprintf("%s-%s", armconst.PrivateEndpointNamePrefix, resourceGroupName))
		if err != nil {
			return nil, err
		}
		log.Debugf("PE IP Address %s ", peIP)
		return &peIP, nil
	}
	return nil, nil
}

func (nm *networkManager) deletePEs(ctx context.Context, resourceName string) error {
	exits := nm.privateEndpointExists(ctx, resourceName)
	if exits {
		_, err := nm.pec.Delete(ctx, nm.managementResourceGroup, resourceName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nm *networkManager) getPrivateEndpointIP(ctx context.Context, resourceName string) (string, error) {
	nic, err := nm.getPrivateEndpointNIC(ctx, resourceName)
	if err != nil {
		return "", err
	}
	for _, ip := range *nic.IPConfigurations {
		if ip.PrivateIPAddress != nil {
			return *ip.PrivateIPAddress, nil
		}
	}

	return "", fmt.Errorf("failed to get private endpoint %s", resourceName)
}

func (nm *networkManager) privateEndpointExists(ctx context.Context, resourceName string) bool {
	nic, err := nm.getPrivateEndpointNIC(ctx, resourceName)
	if err != nil {
		return false
	}
	for _, ip := range *nic.IPConfigurations {
		if ip.PrivateIPAddress != nil {
			return true
		}
	}

	return false
}

func (nm *networkManager) getPrivateEndpointNIC(ctx context.Context, resourceName string) (*aznetwork.Interface, error) {
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
		return &ni, nil
	}

	return nil, fmt.Errorf("no private endpoint found: %s", resourceName)
}
