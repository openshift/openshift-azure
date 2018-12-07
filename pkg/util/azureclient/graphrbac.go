package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
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

// NewRBACApplicationsClient creates a new ApplicationsClient
func NewRBACApplicationsClient(tenantID string, authorizer autorest.Authorizer, languages []string) RBACApplicationsClient {
	client := graphrbac.NewApplicationsClient(tenantID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)
	return &rbacApplicationsClient{
		ApplicationsClient: client,
	}
}
