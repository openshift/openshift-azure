package upgrade

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/initialize"
	"github.com/openshift/openshift-azure/pkg/log"
)

func Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, i initialize.Initializer, azuredeploy []byte) error {
	var t map[string]interface{}
	err := json.Unmarshal(azuredeploy, &t)
	if err != nil {
		return err
	}

	config := auth.NewClientCredentialsConfig(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
	authorizer, err := config.Authorizer()
	if err != nil {
		return err
	}

	dcli := resources.NewDeploymentsClient(cs.Properties.AzProfile.SubscriptionID)
	dcli.Authorizer = authorizer

	log.Info("applying arm template deployment")
	future, err := dcli.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
		Properties: &resources.DeploymentProperties{
			Template: t,
			Mode:     resources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	log.Info("waiting for arm template deployment to complete")
	err = future.WaitForCompletionRef(ctx, dcli.Client)
	if err != nil {
		return err
	}

	return i.InitializeCluster(ctx, cs)
}
