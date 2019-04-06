package fakerp

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/aadapp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

const (
	// IDs originally picked out of thin air
	OSAMasterRoleDefinitionID = "9bc35064-26cf-4536-8e65-40bd22a41071"
	OSAWorkerRoleDefinitionID = "7c1a95fb-9825-4039-b67c-a3644e872c04"
)

type aadManager struct {
	ac  azureclient.RBACApplicationsClient
	sc  azureclient.ServicePrincipalsClient
	rac azureclient.RoleAssignmentsClient
	cs  *api.OpenShiftManagedCluster
}

func newAADManager(ctx context.Context, cs *api.OpenShiftManagedCluster) (*aadManager, error) {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return nil, err
	}

	graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return &aadManager{
		ac:  azureclient.NewRBACApplicationsClient(ctx, cs.Properties.AzProfile.TenantID, graphauthorizer),
		sc:  azureclient.NewServicePrincipalsClient(ctx, cs.Properties.AzProfile.TenantID, graphauthorizer),
		rac: azureclient.NewRoleAssignmentsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer),
		cs:  cs,
	}, nil
}

func (am *aadManager) ensureApp(ctx context.Context, displayName string, p *api.ServicePrincipalProfile, roleDefinitionID string) error {
	if p.ClientID != "" {
		return nil
	}

	p.Secret = uuid.NewV4().String()
	app, err := am.ac.Create(ctx, graphrbac.ApplicationCreateParameters{
		AvailableToOtherTenants: to.BoolPtr(false),
		DisplayName:             &displayName,
		IdentifierUris:          &[]string{"http://localhost/" + uuid.NewV4().String()},
		PasswordCredentials: &[]graphrbac.PasswordCredential{
			{
				Value:   &p.Secret,
				EndDate: &date.Time{Time: time.Now().AddDate(1, 0, 0)},
			},
		},
	})
	if err != nil {
		return err
	}
	p.ClientID = *app.AppID

	sp, err := am.sc.Create(ctx, graphrbac.ServicePrincipalCreateParameters{
		AppID: app.AppID,
	})
	if err != nil {
		return err
	}

	return wait.PollInfinite(5*time.Second, func() (bool, error) {
		_, err = am.rac.Create(ctx, "subscriptions/"+am.cs.Properties.AzProfile.SubscriptionID+"/resourceGroups/"+am.cs.Properties.AzProfile.ResourceGroup, uuid.NewV4().String(), authorization.RoleAssignmentCreateParameters{
			Properties: &authorization.RoleAssignmentProperties{
				RoleDefinitionID: &roleDefinitionID,
				PrincipalID:      sp.ObjectID,
			},
		})
		if err, ok := err.(autorest.DetailedError); ok {
			if err, ok := err.Original.(*azure.RequestError); ok {
				if err.ServiceError.Code == "PrincipalNotFound" {
					return false, nil
				}
			}
		}
		return err == nil, err
	})
}

func (am *aadManager) deleteApp(ctx context.Context, p *api.ServicePrincipalProfile) error {
	objID, err := aadapp.GetApplicationObjectIDFromAppID(ctx, am.ac, p.ClientID)
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
		&am.cs.Properties.MasterServicePrincipalProfile,
		"/subscriptions/"+am.cs.Properties.AzProfile.SubscriptionID+"/providers/Microsoft.Authorization/roleDefinitions/"+OSAMasterRoleDefinitionID)
	if err != nil {
		return err
	}

	return am.ensureApp(ctx, fmt.Sprintf("auto-%d-%s-worker", now, am.cs.Properties.AzProfile.ResourceGroup),
		&am.cs.Properties.WorkerServicePrincipalProfile,
		"/subscriptions/"+am.cs.Properties.AzProfile.SubscriptionID+"/providers/Microsoft.Authorization/roleDefinitions/"+OSAWorkerRoleDefinitionID)
}

func (am *aadManager) deleteApps(ctx context.Context) error {
	err := am.deleteApp(ctx, &am.cs.Properties.MasterServicePrincipalProfile)
	if err != nil {
		return err
	}

	return am.deleteApp(ctx, &am.cs.Properties.WorkerServicePrincipalProfile)
}
