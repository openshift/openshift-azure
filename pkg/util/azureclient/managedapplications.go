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
func NewApplicationsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) ApplicationsClient {
	client := managedapplications.NewApplicationsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)

	return &applicationsClient{
		ApplicationsClient: client,
	}
}

func (c *applicationsClient) Client() autorest.Client {
	return c.ApplicationsClient.Client
}
