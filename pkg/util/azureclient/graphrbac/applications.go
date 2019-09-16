package graphrbac

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE ApplicationsClient,GroupsClient,ServicePrincipalsClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// ApplicationsClient is a minimal interface for azure ApplicationsClient
type ApplicationsClient interface {
	Create(ctx context.Context, parameters graphrbac.ApplicationCreateParameters) (result graphrbac.Application, err error)
	Delete(ctx context.Context, applicationObjectID string) (result autorest.Response, err error)
	Get(ctx context.Context, applicationObjectID string) (result graphrbac.Application, err error)
	List(ctx context.Context, filter string) (result graphrbac.ApplicationListResultPage, err error)
	Patch(ctx context.Context, applicationObjectID string, parameters graphrbac.ApplicationUpdateParameters) (result autorest.Response, err error)
}

type applicationsClient struct {
	graphrbac.ApplicationsClient
}

var _ ApplicationsClient = &applicationsClient{}

// NewApplicationsClient creates a new ApplicationsClient
func NewApplicationsClient(ctx context.Context, log *logrus.Entry, tenantID string, authorizer autorest.Authorizer) ApplicationsClient {
	client := graphrbac.NewApplicationsClient(tenantID)
	azureclient.SetupClient(ctx, log, "graphrbac.ApplicationsClient", &client.Client, authorizer)

	return &applicationsClient{
		ApplicationsClient: client,
	}
}
