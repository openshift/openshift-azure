package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/operationalinsights"
)

// ParseLogAnalyticsResourceID parses this /subscriptions/%s/resourcegroups/defaultresourcegroup-x/providers/microsoft.operationalinsights/workspaces/DefaultWorkspace-x
// and returns the subscription, resourcegroup and workspace name
func ParseLogAnalyticsResourceID(resourceID string) (string, string, string, error) {
	sections := strings.Split(resourceID, "/")
	if len(sections) != 9 {
		return "", "", "", fmt.Errorf("resourceID in the wrong format %s", resourceID)
	}
	return sections[2], sections[4], sections[8], nil
}

func GetWorkspaceInfo(ctx context.Context, client operationalinsights.WorkspacesClient, resourceID string) (string, string, error) {
	_, resourceGroupName, workspaceName, err := ParseLogAnalyticsResourceID(resourceID)
	if err != nil {
		return "", "", err
	}
	w, err := client.Get(ctx, resourceGroupName, workspaceName)
	if err != nil {
		return "", "", err
	}
	if w.WorkspaceProperties == nil || w.WorkspaceProperties.CustomerID == nil {
		return "", "", fmt.Errorf("CustomerID unknown for workspace %s", workspaceName)
	}
	keys, err := client.GetSharedKeys(ctx, resourceGroupName, workspaceName)
	if err != nil {
		return *w.WorkspaceProperties.CustomerID, "", err
	}

	return *w.WorkspaceProperties.CustomerID, *keys.PrimarySharedKey, nil
}
