package fakerp

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func NewAADPasswordCredential() []graphrbac.PasswordCredential {
	azureAadClientSecretID := uuid.NewV4().String()
	azureAadClientSecretValue := uuid.NewV4().String()
	timestart := date.Time{Time: time.Now()}
	timeend := date.Time{Time: timestart.AddDate(1, 0, 0)} // make it valid for a year
	return []graphrbac.PasswordCredential{
		{
			EndDate:   &timeend,
			StartDate: &timestart,
			KeyID:     to.StringPtr(azureAadClientSecretID),
			Value:     to.StringPtr(azureAadClientSecretValue),
		},
	}
}

// UpdateAADAppSecret creates a new secret updates Azure AAD and returns it.
func UpdateAADAppSecret(ctx context.Context, appClient azureclient.RBACApplicationsClient, appID string, callbackURL string) (string, error) {
	pages, err := appClient.List(ctx, fmt.Sprintf("appid eq '%s'", appID))
	if err != nil {
		return "", err
	}
	var apps []graphrbac.Application
	for pages.NotDone() {
		apps = append(apps, pages.Values()...)
		err = pages.Next()
		if err != nil {
			return "", err
		}
	}
	if len(apps) != 1 {
		return "", fmt.Errorf("error: found %d applications, should be 1", len(apps))
	}
	app := apps[0]

	newPc := NewAADPasswordCredential()
	_, err = appClient.Patch(ctx, *app.ObjectID, graphrbac.ApplicationUpdateParameters{
		Homepage:            to.StringPtr(callbackURL),
		ReplyUrls:           &[]string{callbackURL},
		IdentifierUris:      &[]string{callbackURL},
		PasswordCredentials: &newPc,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed patching aad password and uris")
	}
	return *newPc[0].Value, nil
}
