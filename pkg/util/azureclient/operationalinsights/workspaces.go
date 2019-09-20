package operationalinsights

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE WorkspacesClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/operationalinsights/mgmt/2015-11-01-preview/operationalinsights"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// WorkspacesClient is a minimal interface for azure WorkspacesClient
type WorkspacesClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, workspaceName string, parameters operationalinsights.Workspace) (result operationalinsights.WorkspacesCreateOrUpdateFuture, err error)
	Get(ctx context.Context, resourceGroupName string, workspaceName string) (result operationalinsights.Workspace, err error)
	GetSharedKeys(ctx context.Context, resourceGroupName string, workspaceName string) (result operationalinsights.SharedKeys, err error)
}

type workspacesClient struct {
	operationalinsights.WorkspacesClient
}

var _ WorkspacesClient = &workspacesClient{}

// NewWorkspacesClient creates a new WorkspacesClient
func NewWorkspacesClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) WorkspacesClient {
	client := operationalinsights.NewWorkspacesClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "workspaces.WorkspacesClient", &client.Client, authorizer)

	return &workspacesClient{
		WorkspacesClient: client,
	}
}
