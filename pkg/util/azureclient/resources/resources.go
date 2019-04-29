package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// ResourcesClient is a minimal interface for azure Resources Client
type ResourcesClient interface {
	DeleteByID(ctx context.Context, resourceID string) (result resources.DeleteByIDFuture, err error)
	ListByResourceGroup(ctx context.Context, resourceGroupName string, filter string, expand string, top *int32) (result resources.ListResultPage, err error)
}

type resourcesClient struct {
	resources.Client
}

var _ ResourcesClient = &resourcesClient{}

// NewResourcesClient creates a new ResourcesClient
func NewResourcesClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) ResourcesClient {
	client := resources.NewClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "resources.Client", &client.Client, authorizer)

	return &resourcesClient{
		Client: client,
	}
}
