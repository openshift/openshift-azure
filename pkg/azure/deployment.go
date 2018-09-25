package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/openshift/openshift-azure/pkg/log"
)

// DeploymentClient is minimal interface for azure DeploymentClient
type DeploymentClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) (result resources.DeploymentsCreateOrUpdateFuture, err error)
}

// azDeploymentClient implements DeploymentClient.
type azDeploymentClient struct {
	client resources.DeploymentsClient
}

// NewDeploymentClient return DeploymentClient
func NewDeploymentClient(clientID, clientSecret, tenantID, subscriptionID string) (DeploymentClient, error) {

	client := resources.NewDeploymentsClient(subscriptionID)
	config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, err
	}

	client.Authorizer = authorizer
	return &azDeploymentClient{
		client: client,
	}, nil
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
