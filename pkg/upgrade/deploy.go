package upgrade

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func defaultDeployer(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, pluginConfig api.PluginConfig) error {
	var t map[string]interface{}
	err := json.Unmarshal(azuredeploy, &t)
	if err != nil {
		return err
	}

	clients, err := azureclient.NewAzureClients(ctx, cs, pluginConfig)
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
	return future.WaitForCompletionRef(ctx, clients.Deployments.Client)
}

func (u *simpleUpgrader) Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte) error {
	err := u.pluginConfig.Deployer(ctx, cs, azuredeploy, u.pluginConfig)
	if err != nil {
		return err
	}

	err = u.InitializeCluster(ctx, cs)
	if err != nil {
		return err
	}

	return u.postDeployWaitForAll(ctx, cs, map[string]struct{}{})
}
