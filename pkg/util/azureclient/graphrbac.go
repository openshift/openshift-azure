package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

const (
	// https://developer.microsoft.com/en-us/graph/docs/api-reference/beta/api/application_list
	// To list and patch AAD applications, this code needs to have the clientID
	// of an application with the following permissions:
	// API: Windows Azure Active Directory
	//   Delegated permissions:
	//      Sign in and read user profile
	//      Access the directory as the signed-in user
	clientID = "5935b8e2-3915-409c-bfb2-865b7a9291e0"
)

// RBACApplicationsClient is a minimal interface for azure ApplicationsClient
type RBACApplicationsClient interface {
	List(ctx context.Context, filter string) (result graphrbac.ApplicationListResultPage, err error)
	Patch(ctx context.Context, applicationObjectID string, parameters graphrbac.ApplicationUpdateParameters) (result autorest.Response, err error)
}

type rbacApplicationsClient struct {
	graphrbac.ApplicationsClient
}

var _ RBACApplicationsClient = &rbacApplicationsClient{}

// NewAadAuthorizer create authorizer from username/pass
func NewAadAuthorizer(username, password, tenantID string) (autorest.Authorizer, error) {
	c := auth.UsernamePasswordConfig{
		ClientID:    clientID,
		Username:    username,
		Password:    password,
		TenantID:    tenantID,
		AADEndpoint: azure.PublicCloud.ActiveDirectoryEndpoint,
		Resource:    azure.PublicCloud.GraphEndpoint,
	}
	return c.Authorizer()
}

// NewRBACApplicationsClient creates a new ApplicationsClient
func NewRBACApplicationsClient(tenantID string, authorizer autorest.Authorizer, languages []string) RBACApplicationsClient {
	client := graphrbac.NewApplicationsClient(tenantID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)
	return &rbacApplicationsClient{
		ApplicationsClient: client,
	}
}
