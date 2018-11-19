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
	Create(ctx context.Context, parameters graphrbac.ApplicationCreateParameters) (result graphrbac.Application, err error)
	Delete(ctx context.Context, applicationObjectID string) (result autorest.Response, err error)
	List(ctx context.Context, filter string) (result graphrbac.ApplicationListResultPage, err error)
	Get(ctx context.Context, applicationObjectID string) (result graphrbac.Application, err error)
	ListPasswordCredentials(ctx context.Context, applicationObjectID string) (result graphrbac.PasswordCredentialListResult, err error)
	Patch(ctx context.Context, applicationObjectID string, parameters graphrbac.ApplicationUpdateParameters) (result autorest.Response, err error)
	UpdatePasswordCredentials(ctx context.Context, applicationObjectID string, parameters graphrbac.PasswordCredentialsUpdateParameters) (result autorest.Response, err error)
	Client
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

func (c *rbacApplicationsClient) Create(ctx context.Context, parameters graphrbac.ApplicationCreateParameters) (result graphrbac.Application, err error) {
	return c.ApplicationsClient.Create(ctx, parameters)
}

func (c *rbacApplicationsClient) Delete(ctx context.Context, applicationObjectID string) (result autorest.Response, err error) {
	return c.ApplicationsClient.Delete(ctx, applicationObjectID)
}

func (c *rbacApplicationsClient) Get(ctx context.Context, applicationObjectID string) (result graphrbac.Application, err error) {
	return c.ApplicationsClient.Get(ctx, applicationObjectID)
}

func (c *rbacApplicationsClient) ListPasswordCredentials(ctx context.Context, applicationObjectID string) (result graphrbac.PasswordCredentialListResult, err error) {
	return c.ApplicationsClient.ListPasswordCredentials(ctx, applicationObjectID)
}

func (c *rbacApplicationsClient) List(ctx context.Context, filter string) (result graphrbac.ApplicationListResultPage, err error) {
	return c.ApplicationsClient.List(ctx, filter)
}

func (c *rbacApplicationsClient) Patch(ctx context.Context, applicationObjectID string, parameters graphrbac.ApplicationUpdateParameters) (result autorest.Response, err error) {
	return c.ApplicationsClient.Patch(ctx, applicationObjectID, parameters)
}

func (c *rbacApplicationsClient) UpdatePasswordCredentials(ctx context.Context, applicationObjectID string, parameters graphrbac.PasswordCredentialsUpdateParameters) (result autorest.Response, err error) {
	return c.ApplicationsClient.UpdatePasswordCredentials(ctx, applicationObjectID, parameters)
}

func (c *rbacApplicationsClient) Client() autorest.Client {
	return c.ApplicationsClient.Client
}
