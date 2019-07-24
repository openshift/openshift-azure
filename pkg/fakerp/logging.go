package fakerp

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/operationalinsights/mgmt/2015-11-01-preview/operationalinsights"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func newWorkspaceClient(ctx context.Context, subscriptionID string) (*operationalinsights.WorkspacesClient, error) {
	wc := operationalinsights.NewWorkspacesClient(subscriptionID)
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}
	wc.Authorizer = authorizer
	return &wc, nil
}

// /subscriptions/%s/resourcegroups/defaultresourcegroup-x/providers/microsoft.operationalinsights/workspaces/DefaultWorkspace-x
func parseLogAnalyticsResourceID(resourceID string) (string, string, string, error) {
	sections := strings.Split(resourceID, "/")
	if len(sections) != 9 {
		return "", "", "", fmt.Errorf("resourceID in the wrong format %s", resourceID)
	}
	return sections[2], sections[4], sections[8], nil
}

func getWorkspaceInfo(ctx context.Context, subscriptionID, resourceID string) (string, string, error) {
	wc, err := newWorkspaceClient(ctx, subscriptionID)
	if err != nil {
		return "", "", err
	}
	subID, resourceGroupName, workspaceName, err := parseLogAnalyticsResourceID(resourceID)
	if err != nil {
		return "", "", err
	}
	if subID != subscriptionID {
		return "", "", fmt.Errorf("workspace is in a different subscription %s != %s", subID, subscriptionID)
	}
	w, err := wc.Get(ctx, resourceGroupName, workspaceName)
	if err != nil {
		return "", "", err
	}
	if w.WorkspaceProperties == nil || w.WorkspaceProperties.CustomerID == nil {
		return "", "", fmt.Errorf("CustomerID unknown for workspace %s", workspaceName)
	}
	keys, err := wc.GetSharedKeys(ctx, resourceGroupName, workspaceName)
	if err != nil {
		return *w.WorkspaceProperties.CustomerID, "", err
	}

	return *w.WorkspaceProperties.CustomerID, *keys.PrimarySharedKey, nil
}
