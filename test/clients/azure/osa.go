package azure

import (
	"context"
	"fmt"
	"strings"
)

// OSAResourceGroup returns the name of the resource group holding the OSA
// cluster resources
func (cli *Client) OSAResourceGroup(ctx context.Context, resourcegroup, name, location string) (string, error) {
	appName := strings.Join([]string{"OS", resourcegroup, name, location}, "_")

	app, err := cli.Applications.Get(ctx, appName, appName)
	if err != nil {
		return "", err
	}
	if app.ApplicationProperties == nil {
		return "", fmt.Errorf("managed application %q not found", appName)
	}

	// can't use azure.ParseResourceID here because rgid is of the short form
	// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}
	rgid := *app.ApplicationProperties.ManagedResourceGroupID
	return rgid[strings.LastIndexByte(rgid, '/')+1:], nil
}
