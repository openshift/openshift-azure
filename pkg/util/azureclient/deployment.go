package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

func NewDeploymentsClient(subscriptionID string, authorizer autorest.Authorizer, pluginConfig api.PluginConfig) DeploymentClient {
	client := resources.NewDeploymentsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)
	return &azDeploymentClient{
		client: client,
	}
}

// CreateOrUpdate creates or updates azure depployment
func (az azDeploymentClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) (result resources.DeploymentsCreateOrUpdateFuture, err error) {
	future, err := az.client.CreateOrUpdate(ctx, resourceGroupName, deploymentName, parameters)
	if err != nil {
		return result, err
	}
	log.Info("waiting for arm template deployment to complete")
	err = future.WaitForCompletionRef(ctx, az.client.Client)
	if err != nil {
		return future, err
	}
	return future, err
}
