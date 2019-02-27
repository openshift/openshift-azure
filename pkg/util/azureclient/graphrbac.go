package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
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
func NewRBACApplicationsClient(ctx context.Context, tenantID string, authorizer autorest.Authorizer) RBACApplicationsClient {
	client := graphrbac.NewApplicationsClient(tenantID)
	setupClient(ctx, &client.Client, authorizer)

	return &rbacApplicationsClient{
		ApplicationsClient: client,
	}
}

type ServicePrincipalsClient interface {
	List(ctx context.Context, filter string) (graphrbac.ServicePrincipalListResultPage, error)
}

type servicePrincipalsClient struct {
	graphrbac.ServicePrincipalsClient
}

var _ ServicePrincipalsClient = &servicePrincipalsClient{}

// NewServicePrincipalsClient create a client to query ServicePrincipal information
func NewServicePrincipalsClient(cfg auth.ClientCredentialsConfig, tenantID string) (ServicePrincipalsClient, error) {
	var err error
	var spc servicePrincipalsClient
	spc.ServicePrincipalsClient = graphrbac.NewServicePrincipalsClient(tenantID)
	cfg.Resource = azure.PublicCloud.GraphEndpoint
	spc.Authorizer, err = cfg.Authorizer()
	return &spc, err
}
