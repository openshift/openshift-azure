package upgrade

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/azure"
	"github.com/openshift/openshift-azure/pkg/initialize"
	"github.com/openshift/openshift-azure/pkg/log"
)

func Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, i initialize.Initializer, azuredeploy []byte) error {
	var t map[string]interface{}
	err := json.Unmarshal(azuredeploy, &t)
	if err != nil {
		return err
	}

	client, err := azure.NewDeploymentClient(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string), cs.Properties.AzProfile.SubscriptionID)
	if err != nil {
		return err
	}
	log.Info("applying arm template deployment")
	_, err = client.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
		Properties: &resources.DeploymentProperties{
			Template: t,
			Mode:     resources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	return i.InitializeCluster(ctx, cs)
}
