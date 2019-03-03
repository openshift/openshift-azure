package aadapp

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

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

// UpdateAADApp updates the ReplyURLs in an AAD app.  A side-effect is that the
// secret must be regenerated.  The new secret is returned.
func UpdateAADApp(ctx context.Context, appClient azureclient.RBACApplicationsClient, appObjID string, callbackURL string) (string, error) {
	azureAadClientSecretID := uuid.NewV4().String()
	azureAadClientSecretValue := uuid.NewV4().String()

	// Create a new password credential
	timestart := date.Time{Time: time.Now()}
	timeend := date.Time{Time: timestart.AddDate(1, 0, 0)} // make it valid for a year
	newPc := []graphrbac.PasswordCredential{
		{
			EndDate:   &timeend,
			StartDate: &timestart,
			KeyID:     to.StringPtr(azureAadClientSecretID),
			Value:     to.StringPtr(azureAadClientSecretValue),
		},
	}
	_, err := appClient.Patch(ctx, appObjID, graphrbac.ApplicationUpdateParameters{
		Homepage:            to.StringPtr(callbackURL),
		ReplyUrls:           &[]string{callbackURL},
		IdentifierUris:      &[]string{callbackURL},
		PasswordCredentials: &newPc,
	})
	if err != nil {
		return "", fmt.Errorf("failed patching aad password and uris: %v", err)
	}
	return azureAadClientSecretValue, nil
}
