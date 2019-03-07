package azureclient

import (
	"context"
)

// NOTE: we should be very sparing and have a high bar for these kind of hacks.

type ZonesClientAddons interface {
	Delete(ctx context.Context, resourceGroupName string, zoneName string, ifMatch string) error
}

func (c *zonesClient) Delete(ctx context.Context, resourceGroupName string, zoneName string, ifMatch string) error {
	future, err := c.ZonesClient.Delete(ctx, resourceGroupName, zoneName, ifMatch)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.ZonesClient.Client)
}
