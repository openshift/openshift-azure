package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"

	"github.com/openshift/openshift-azure/pkg/api"
)

// NewDeploymentsClient creates a new deployments client
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
	return az.client.CreateOrUpdate(ctx, resourceGroupName, deploymentName, parameters)
}

func (az azDeploymentClient) GetClient() autorest.Client {
	return az.client.Client
}
