package fakerp

import (
	"context"
	"fmt"
	"strings"
	"time"

	azauthorization "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/aadapp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/authorization"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/graphrbac"
)

const (
	// IDs originally picked out of thin air
	OSAMasterRoleDefinitionID = "9bc35064-26cf-4536-8e65-40bd22a41071"
	OSAWorkerRoleDefinitionID = "7c1a95fb-9825-4039-b67c-a3644e872c04"
)

type aadManager struct {
	testConfig api.TestConfig
	ac         graphrbac.ApplicationsClient
	sc         graphrbac.ServicePrincipalsClient
	rac        authorization.RoleAssignmentsClient
	cs         *api.OpenShiftManagedCluster
	log        *logrus.Entry
}

func newAADManager(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) (*aadManager, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	graphauthorizer, err := azureclient.GetAuthorizerFromContext(ctx, contextKeyGraphClientAuthorizer)
	if err != nil {
		return nil, err
	}

	return &aadManager{
		testConfig: testConfig,
		ac:         graphrbac.NewApplicationsClient(ctx, log, cs.Properties.AzProfile.TenantID, graphauthorizer),
		sc:         graphrbac.NewServicePrincipalsClient(ctx, log, cs.Properties.AzProfile.TenantID, graphauthorizer),
		rac:        authorization.NewRoleAssignmentsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer),
		cs:         cs,
		log:        log,
	}, nil
}

func (am *aadManager) ensureApp(ctx context.Context, displayName string, clientID *string, secret *string, roleDefinitionID string) error {
	if *clientID != "" {
		return nil
	}
	am.log.Debugf("create aad app %s", displayName)
	*secret = uuid.NewV4().String()
	appParam := azgraphrbac.ApplicationCreateParameters{
		AvailableToOtherTenants: to.BoolPtr(false),
		DisplayName:             &displayName,
		IdentifierUris:          &[]string{"http://localhost/" + uuid.NewV4().String()},
		PasswordCredentials: &[]azgraphrbac.PasswordCredential{
			{
				Value:   secret,
				EndDate: &date.Time{Time: time.Now().AddDate(1, 0, 0)},
			},
		},
	}

	app, err := am.ac.Create(ctx, appParam)
	if err != nil {
		return err
	}
	*clientID = *app.AppID

	var sp azgraphrbac.ServicePrincipal
	err = wait.PollInfinite(5*time.Second, func() (bool, error) {
		sp, err = am.sc.Create(ctx, azgraphrbac.ServicePrincipalCreateParameters{
			AppID: app.AppID,
		})
		// ugh: Azure client library doesn't have the types registered to
		// unmarshal all the way down to this error code natively :-(
		if err != nil && strings.Contains(err.Error(), "NoBackingApplicationObject") {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return err
	}

	err = wait.PollInfinite(5*time.Second, func() (bool, error) {
		_, err = am.rac.Create(ctx, "subscriptions/"+am.cs.Properties.AzProfile.SubscriptionID+"/resourceGroups/"+am.cs.Properties.AzProfile.ResourceGroup, uuid.NewV4().String(), azauthorization.RoleAssignmentCreateParameters{
			Properties: &azauthorization.RoleAssignmentProperties{
				RoleDefinitionID: &roleDefinitionID,
				PrincipalID:      sp.ObjectID,
			},
		})
		if err, ok := err.(autorest.DetailedError); ok {
			if err, ok := err.Original.(*azure.RequestError); ok {
				if err.ServiceError != nil && err.ServiceError.Code == "PrincipalNotFound" {
					return false, nil
				}
			}
		}
		return err == nil, err
	})
	if err != nil {
		return err
	}

	if am.testConfig.ImageResourceName != "" {
		// needed for the e2e lb test when running from an image
		_, err = am.rac.Create(ctx, "subscriptions/"+am.cs.Properties.AzProfile.SubscriptionID+"/resourceGroups/"+am.testConfig.ImageResourceGroup+"/providers/Microsoft.Compute/images/"+am.testConfig.ImageResourceName, uuid.NewV4().String(), azauthorization.RoleAssignmentCreateParameters{
			Properties: &azauthorization.RoleAssignmentProperties{
				RoleDefinitionID: to.StringPtr("/subscriptions/" + am.cs.Properties.AzProfile.SubscriptionID + "/providers/Microsoft.Authorization/roleDefinitions/acdd72a7-3385-48ef-bd42-f606fba81ae7"), // Reader
				PrincipalID:      sp.ObjectID,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (am *aadManager) deleteApp(ctx context.Context, clientID *string) error {
	objID, err := aadapp.GetApplicationObjectIDFromAppID(ctx, am.ac, *clientID)
	if err != nil {
		return err
	}

	_, err = am.ac.Delete(ctx, objID)
	return err

	// deleting the app automatically deletes the service principal
}

func (am *aadManager) ensureApps(ctx context.Context) error {
	now := time.Now().Unix()

	err := am.ensureApp(ctx, fmt.Sprintf("auto-%d-%s-master", now, am.cs.Properties.AzProfile.ResourceGroup),
		&am.cs.Properties.MasterServicePrincipalProfile.ClientID, &am.cs.Properties.MasterServicePrincipalProfile.Secret,
		"/subscriptions/"+am.cs.Properties.AzProfile.SubscriptionID+"/providers/Microsoft.Authorization/roleDefinitions/"+OSAMasterRoleDefinitionID)
	if err != nil {
		return err
	}

	return am.ensureApp(ctx, fmt.Sprintf("auto-%d-%s-worker", now, am.cs.Properties.AzProfile.ResourceGroup),
		&am.cs.Properties.WorkerServicePrincipalProfile.ClientID, &am.cs.Properties.WorkerServicePrincipalProfile.Secret,
		"/subscriptions/"+am.cs.Properties.AzProfile.SubscriptionID+"/providers/Microsoft.Authorization/roleDefinitions/"+OSAWorkerRoleDefinitionID)
}

func (am *aadManager) deleteApps(ctx context.Context) error {
	err := am.deleteApp(ctx, &am.cs.Properties.MasterServicePrincipalProfile.ClientID)
	if err != nil {
		return err
	}

	return am.deleteApp(ctx, &am.cs.Properties.WorkerServicePrincipalProfile.ClientID)
}
