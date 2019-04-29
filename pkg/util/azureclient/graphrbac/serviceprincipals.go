package graphrbac

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type ServicePrincipalsClient interface {
	Create(ctx context.Context, parameters graphrbac.ServicePrincipalCreateParameters) (result graphrbac.ServicePrincipal, err error)
	List(ctx context.Context, filter string) (graphrbac.ServicePrincipalListResultPage, error)
}

type servicePrincipalsClient struct {
	graphrbac.ServicePrincipalsClient
}

var _ ServicePrincipalsClient = &servicePrincipalsClient{}

// NewServicePrincipalsClient create a client to query ServicePrincipal information
func NewServicePrincipalsClient(ctx context.Context, log *logrus.Entry, tenantID string, authorizer autorest.Authorizer) ServicePrincipalsClient {
	client := graphrbac.NewServicePrincipalsClient(tenantID)
	azureclient.SetupClient(ctx, log, "graphrbac.ServicePrincipalsClient", &client.Client, authorizer)

	return &servicePrincipalsClient{
		ServicePrincipalsClient: client,
	}
}
