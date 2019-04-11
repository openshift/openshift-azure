package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
)

type DeploymentsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) error
}

func (c *deploymentsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) error {
	future, err := c.DeploymentsClient.CreateOrUpdate(ctx, resourceGroupName, deploymentName, parameters)
	if err != nil {
		return err
	}
	return future.WaitForCompletionRef(ctx, c.DeploymentsClient.Client)
}

type DeploymentOperationsClientAddons interface {
	List(ctx context.Context, resourceGroupName string, deploymentName string, top *int32) ([]resources.DeploymentOperation, error)
}

func (c *deploymentOperationsClient) List(ctx context.Context, resourceGroupName string, deploymentName string, top *int32) ([]resources.DeploymentOperation, error) {
	pages, err := c.DeploymentOperationsClient.List(ctx, resourceGroupName, deploymentName, top)
	if err != nil {
		return nil, err
	}

	var operations []resources.DeploymentOperation
	for pages.NotDone() {
		operations = append(operations, pages.Values()...)

		err = pages.Next()
		if err != nil {
			return nil, err
		}
	}

	return operations, nil
}
