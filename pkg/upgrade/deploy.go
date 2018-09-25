package upgrade

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/initialize"
	"github.com/openshift/openshift-azure/pkg/log"
)

func Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, i initialize.Initializer, azuredeploy []byte, config api.PluginConfig) error {
	var t map[string]interface{}
	err := json.Unmarshal(azuredeploy, &t)
	if err != nil {
		return err
	}

	clients, err := azureclient.NewAzureClients(ctx, cs, config)
	if err != nil {
		return err
	}

	log.Info("applying arm template deployment")
	future, err := clients.Deployments.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
		Properties: &resources.DeploymentProperties{
			Template: t,
			Mode:     resources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	log.Info("waiting for arm template deployment to complete")
	err = future.WaitForCompletionRef(ctx, clients.Deployments.Client)
	if err != nil {
		return err
	}

	return i.InitializeCluster(ctx, cs)
}
