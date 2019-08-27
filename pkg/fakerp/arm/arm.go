package arm

import (
	"context"

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
	return arm.Generate(ctx, cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, resource)
}

// GenerateRPSide generates fakeRP callback function for RP side objects. This function mocks realRP
// impementation for required objects
func GenerateRPSide(ctx context.Context, cs *api.OpenShiftManagedCluster, conf *client.Config, now int64) (map[string]interface{}, error) {
	resource := []interface{}{
		privateEndpoint(cs, conf, now),
	}
	return arm.Generate(ctx, cs.Properties.AzProfile.SubscriptionID, conf.ManagementResourceGroup, resource)
}
