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

// GetObjectIDUsingRbacClient find the ObjectID for the application
func GetObjectIDUsingRbacClient(ctx context.Context, appClient azureclient.RBACApplicationsClient, appID string) (string, error) {
	pages, err := appClient.List(ctx, fmt.Sprintf("appid eq '%s'", appID))
	if err != nil {
		return "", fmt.Errorf("failed listing applications: %v", err)
	}
	apps := pages.Values()
	if len(apps) != 1 {
		return "", fmt.Errorf("found %d applications, should be 1", len(apps))
	}
	return *apps[0].ObjectID, nil
}

// GetObjectIDUsingSPClient find the ObjectID for the application
func GetObjectIDUsingSPClient(ctx context.Context, spc azureclient.ServicePrincipalsClient, appID string) (string, error) {
	sp, err := spc.List(ctx, fmt.Sprintf("appID eq '%s'", appID))
	if err != nil {
		return "", err
	}

	if len(sp.Values()) != 1 {
		return "", fmt.Errorf("graph query returned %d values", len(sp.Values()))
	}

	return *sp.Values()[0].ObjectID, nil
}

// UpdateSecret creates a new secret updates Azure AAD and returns it.
func UpdateSecret(ctx context.Context, appClient azureclient.RBACApplicationsClient, appObjID string, callbackURL string) (string, error) {
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
