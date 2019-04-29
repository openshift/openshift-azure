package managedapplications

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE ApplicationsClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/preview/resources/mgmt/managedapplications"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// ApplicationsClient is a minimal interface for azure ApplicationsClient
type ApplicationsClient interface {
	Get(ctx context.Context, resourceGroupName string, applicationName string) (result managedapplications.Application, err error)
	GetByID(ctx context.Context, applicationID string) (result managedapplications.Application, err error)
	ListByResourceGroup(ctx context.Context, resourceGroupName string) (result managedapplications.ApplicationListResultPage, err error)
	azureclient.Client
}

type applicationsClient struct {
	managedapplications.ApplicationsClient
}

var _ ApplicationsClient = &applicationsClient{}

// NewApplicationsClient creates a new ApplicationsClient
func NewApplicationsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) ApplicationsClient {
	client := managedapplications.NewApplicationsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "managedapplications.ApplicationsClient", &client.Client, authorizer)

	return &applicationsClient{
		ApplicationsClient: client,
	}
}

func (c *applicationsClient) Client() autorest.Client {
	return c.ApplicationsClient.Client
}
