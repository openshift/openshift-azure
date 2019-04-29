package authorization

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type RoleDefinitionsClient interface {
	CreateOrUpdate(ctx context.Context, scope string, roleDefinitionID string, roleDefinition authorization.RoleDefinition) (result authorization.RoleDefinition, err error)
}

type roleDefinitionsClient struct {
	authorization.RoleDefinitionsClient
}

var _ RoleDefinitionsClient = &roleDefinitionsClient{}

// NewRoleDefinitionsClient creates a new RoleDefinitionsClient
func NewRoleDefinitionsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) RoleDefinitionsClient {
	client := authorization.NewRoleDefinitionsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "authorization.RoleDefinitionsClient", &client.Client, authorizer)

	return &roleDefinitionsClient{
		RoleDefinitionsClient: client,
	}
}
