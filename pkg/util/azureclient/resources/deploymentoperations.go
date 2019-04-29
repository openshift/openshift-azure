package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// DeploymentOperationsClient is a minimal interface for azure DeploymentOperationsClient
type DeploymentOperationsClient interface {
	DeploymentOperationsClientAddons
}

type deploymentOperationsClient struct {
	resources.DeploymentOperationsClient
}

var _ DeploymentOperationsClient = &deploymentOperationsClient{}

// NewDeploymentOperationsClient creates a new DeploymentOperationsClient
func NewDeploymentOperationsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) DeploymentOperationsClient {
	client := resources.NewDeploymentOperationsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "resources.DeploymentOperationsClient", &client.Client, authorizer)

	return &deploymentOperationsClient{
		DeploymentOperationsClient: client,
	}
}
