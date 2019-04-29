package graphrbac

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE RBACApplicationsClient,RBACGroupsClient,ServicePrincipalsClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// RBACApplicationsClient is a minimal interface for azure ApplicationsClient
type RBACApplicationsClient interface {
	Create(ctx context.Context, parameters graphrbac.ApplicationCreateParameters) (result graphrbac.Application, err error)
	Delete(ctx context.Context, applicationObjectID string) (result autorest.Response, err error)
	List(ctx context.Context, filter string) (result graphrbac.ApplicationListResultPage, err error)
	Patch(ctx context.Context, applicationObjectID string, parameters graphrbac.ApplicationUpdateParameters) (result autorest.Response, err error)
}

type rbacApplicationsClient struct {
	graphrbac.ApplicationsClient
}

var _ RBACApplicationsClient = &rbacApplicationsClient{}

// NewRBACApplicationsClient creates a new ApplicationsClient
func NewRBACApplicationsClient(ctx context.Context, log *logrus.Entry, tenantID string, authorizer autorest.Authorizer) RBACApplicationsClient {
	client := graphrbac.NewApplicationsClient(tenantID)
	azureclient.SetupClient(ctx, log, "graphrbac.ApplicationsClient", &client.Client, authorizer)

	return &rbacApplicationsClient{
		ApplicationsClient: client,
	}
}
