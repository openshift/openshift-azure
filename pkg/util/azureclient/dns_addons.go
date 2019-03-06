package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
)

// NOTE: we should be very sparing and have a high bar for these kind of hacks.

type ZonesClientAddons interface {
	Delete(ctx context.Context, resourceGroupName string, zoneName string, ifMatch string) error
	ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) ([]dns.Zone, error)
}

func (c *zonesClient) Delete(ctx context.Context, resourceGroupName string, zoneName string, ifMatch string) error {
	future, err := c.ZonesClient.Delete(ctx, resourceGroupName, zoneName, ifMatch)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.ZonesClient.Client)
}

func (c *zonesClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, top *int32) ([]dns.Zone, error) {
	pages, err := c.ZonesClient.ListByResourceGroup(ctx, resourceGroupName, top)
	if err != nil {
		return nil, err
	}

	var zones []dns.Zone
	for pages.NotDone() {
		zones = append(zones, pages.Values()...)

		err = pages.Next()
		if err != nil {
			return nil, err
		}
	}

	return zones, nil
}
