package graphrbac

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// RBACGroupsClient is a minimal interface for azure GroupsClient
type RBACGroupsClient interface {
	RBACGroupsClientAddons
}

type rbacGroupsClient struct {
	graphrbac.GroupsClient
}

var _ RBACGroupsClient = &rbacGroupsClient{}

// NewRBACApplicationsClient creates a new ApplicationsClient
func NewRBACGroupsClient(ctx context.Context, log *logrus.Entry, tenantID string, authorizer autorest.Authorizer) RBACGroupsClient {
	client := graphrbac.NewGroupsClient(tenantID)
	azureclient.SetupClient(ctx, log, "graphrbac.GroupsClient", &client.Client, authorizer)

	return &rbacGroupsClient{
		GroupsClient: client,
	}
}
