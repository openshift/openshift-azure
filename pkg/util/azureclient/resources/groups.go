package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// GroupsClient is a minimal interface for azure Resources Client
type GroupsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, parameters resources.Group) (result resources.Group, err error)
	List(ctx context.Context, filter string, top *int32) (result resources.GroupListResultPage, err error)
	Delete(ctx context.Context, resourceGroupName string) (result resources.GroupsDeleteFuture, err error)
	azureclient.Client
}

type groupsClient struct {
	resources.GroupsClient
}

var _ GroupsClient = &groupsClient{}

// NewGroupsClient creates a new ResourcesClient
func NewGroupsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) GroupsClient {
	client := resources.NewGroupsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "resources.GroupsClient", &client.Client, authorizer)

	return &groupsClient{
		GroupsClient: client,
	}
}

func (c *groupsClient) Client() autorest.Client {
	return c.GroupsClient.Client
}
