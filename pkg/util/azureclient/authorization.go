package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"
)

type RoleAssignmentsClient interface {
	Create(ctx context.Context, scope string, roleAssignmentName string, parameters authorization.RoleAssignmentCreateParameters) (result authorization.RoleAssignment, err error)
}

type roleAssignmentsClient struct {
	authorization.RoleAssignmentsClient
}

var _ RoleAssignmentsClient = &roleAssignmentsClient{}

// NewRoleAssignmentsClient creates a new RoleAssignmentsClient
func NewRoleAssignmentsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) RoleAssignmentsClient {
	client := authorization.NewRoleAssignmentsClient(subscriptionID)
	setupClient(ctx, log, "authorization.RoleAssignmentsClient", &client.Client, authorizer)

	return &roleAssignmentsClient{
		RoleAssignmentsClient: client,
	}
}

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
	setupClient(ctx, log, "authorization.RoleDefinitionsClient", &client.Client, authorizer)

	return &roleDefinitionsClient{
		RoleDefinitionsClient: client,
	}
}
