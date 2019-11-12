package arm

import (
	"context"
	"encoding/json"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/util/arm"
)

// GenerateClusterSide generates fakeRP callback function objects for cluster side objects. This function mocks realRP
// impementation for required objects
func GenerateClusterSide(ctx context.Context, cs *api.OpenShiftManagedCluster) (map[string]interface{}, error) {
	resource := []interface{}{
		privateLinkService(cs),
	}
	return Generate(ctx, cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, resource)
}

// GenerateRPSide generates fakeRP callback function for RP side objects. This function mocks realRP
// impementation for required objects
func GenerateRPSide(ctx context.Context, cs *api.OpenShiftManagedCluster, conf *client.Config) (map[string]interface{}, error) {
	resource := []interface{}{
		privateEndpoint(cs, conf),
	}
	return Generate(ctx, cs.Properties.AzProfile.SubscriptionID, conf.ManagementResourceGroup, resource)
}

// Generate generates ARM template, based on resource interface provided
// This version of Generate will not do any version fixup as it is used
// in fakeRP implementation
func Generate(ctx context.Context, subscriptionID, resourceGroup string, resource []interface{}) (map[string]interface{}, error) {
	t := arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resource,
	}

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var azuretemplate map[string]interface{}
	err = json.Unmarshal(b, &azuretemplate)
	if err != nil {
		return nil, err
	}

	arm.FixupDepends(subscriptionID, resourceGroup, azuretemplate, nil)

	return azuretemplate, nil
}
