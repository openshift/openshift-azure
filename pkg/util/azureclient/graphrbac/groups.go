package graphrbac

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// GroupsClient is a minimal interface for azure GroupsClient
type GroupsClient interface {
	GroupsClientAddons
}

type groupsClient struct {
	graphrbac.GroupsClient
}

var _ GroupsClient = &groupsClient{}

// NewApplicationsClient creates a new ApplicationsClient
func NewGroupsClient(ctx context.Context, log *logrus.Entry, tenantID string, authorizer autorest.Authorizer) GroupsClient {
	client := graphrbac.NewGroupsClient(tenantID)
	azureclient.SetupClient(ctx, log, "graphrbac.GroupsClient", &client.Client, authorizer)

	return &groupsClient{
		GroupsClient: client,
	}
}
