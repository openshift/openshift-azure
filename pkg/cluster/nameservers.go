package cluster

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/arm/constants"
)

// GetNameserversFromVnet return the nameservers configured in the vnet or the default
func (u *Upgrade) GetNameserversFromVnet(ctx context.Context, log *logrus.Entry, subscriptionID, resourceGroup string) ([]string, error) {
	vnet, err := u.Vnc.Get(ctx, resourceGroup, constants.VnetName, "")
	if err != nil {
		return nil, err
	}
	if vnet.VirtualNetworkPropertiesFormat != nil && vnet.VirtualNetworkPropertiesFormat.DhcpOptions != nil && vnet.VirtualNetworkPropertiesFormat.DhcpOptions.DNSServers != nil {
		return *vnet.VirtualNetworkPropertiesFormat.DhcpOptions.DNSServers, nil
	}
	return []string{constants.AzureNameserver}, nil
}
