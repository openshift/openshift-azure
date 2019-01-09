package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/preview/resources/mgmt/managedapplications"
	"github.com/Azure/go-autorest/autorest"
)

// ApplicationsClient is a minimal interface for azure ApplicationsClient
type ApplicationsClient interface {
	Get(ctx context.Context, resourceGroupName string, applicationName string) (result managedapplications.Application, err error)
	GetByID(ctx context.Context, applicationID string) (result managedapplications.Application, err error)
	ListByResourceGroup(ctx context.Context, resourceGroupName string) (result managedapplications.ApplicationListResultPage, err error)
	Client
}

type applicationsClient struct {
	managedapplications.ApplicationsClient
}

var _ ApplicationsClient = &applicationsClient{}

// NewApplicationsClient creates a new ApplicationsClient
func NewApplicationsClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) ApplicationsClient {
	client := managedapplications.NewApplicationsClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

	return &applicationsClient{
		ApplicationsClient: client,
	}
}

func (c *applicationsClient) Client() autorest.Client {
	return c.ApplicationsClient.Client
}
