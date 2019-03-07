package aadapp

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// GetApplicationObjectIDFromAppID returns the ObjectID of the AAD application
// corresponding to a given appID
func GetApplicationObjectIDFromAppID(ctx context.Context, appClient azureclient.RBACApplicationsClient, appID string) (string, error) {
	app, err := appClient.List(ctx, fmt.Sprintf("appid eq '%s'", appID))
	if err != nil {
		return "", err
	}

	if len(app.Values()) != 1 {
		return "", fmt.Errorf("found %d applications, should be 1", len(app.Values()))
	}

	return *app.Values()[0].ObjectID, nil
}

// GetServicePrincipalObjectIDFromAppID returns the ObjectID of the service
// principal corresponding to a given appID
func GetServicePrincipalObjectIDFromAppID(ctx context.Context, spc azureclient.ServicePrincipalsClient, appID string) (string, error) {
	sp, err := spc.List(ctx, fmt.Sprintf("appID eq '%s'", appID))
	if err != nil {
		return "", err
	}

	if len(sp.Values()) != 1 {
		return "", fmt.Errorf("found %d service principals, should be 1", len(sp.Values()))
	}

	return *sp.Values()[0].ObjectID, nil
}

// UpdateAADApp updates the ReplyURLs in an AAD app.
func UpdateAADApp(ctx context.Context, appClient azureclient.RBACApplicationsClient, appObjID string, callbackURL string) error {
	_, err := appClient.Patch(ctx, appObjID, graphrbac.ApplicationUpdateParameters{
		Homepage:       to.StringPtr(callbackURL),
		ReplyUrls:      &[]string{callbackURL},
		IdentifierUris: &[]string{callbackURL},
	})
	return err
}
