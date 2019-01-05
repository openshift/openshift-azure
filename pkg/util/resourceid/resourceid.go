package resourceid

import "fmt"

// ResourceID constructs an Azure resource ID offline from the provided components
func ResourceID(subscriptionID, resourceGroup, resourceProvider, resourceName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", subscriptionID, resourceGroup, resourceProvider, resourceName)
}
