package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/authorization"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/graphrbac"
)

const (
	aroTeamSharedID = "8b45c9d6-0b66-44ef-93df-7a6b0c4c1635"
)

type aadManager struct {
	log *logrus.Entry
	ac  graphrbac.ApplicationsClient
	sc  graphrbac.ServicePrincipalsClient
	rac authorization.RoleAssignmentsClient
}

func newAADManager(ctx context.Context, log *logrus.Entry) (*aadManager, error) {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return nil, err
	}

	graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return &aadManager{
		log: log,
		ac:  graphrbac.NewApplicationsClient(ctx, log, os.Getenv("AZURE_TENANT_ID"), graphauthorizer),
		sc:  graphrbac.NewServicePrincipalsClient(ctx, log, os.Getenv("AZURE_TENANT_ID"), graphauthorizer),
		rac: authorization.NewRoleAssignmentsClient(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
	}, nil
}

// rotateSecret ensures that all ARO dev secrets are rotated
func (am *aadManager) rotateSecret(ctx context.Context) error {
	am.log.Info("rotatesecrets")
	results, err := am.ac.List(ctx, "")
	if err != nil {
		return err
	}

	passwords := []azgraphrbac.PasswordCredential{}
	for ; results.NotDone(); results.Next() {
		for _, app := range results.Values() {
			id := *app.ObjectID
			if strings.EqualFold(id, aroTeamSharedID) {
				for _, passwd := range *app.PasswordCredentials {
					passwords = append(passwords, passwd)
				}
			}
		}
	}

	secret := uuid.NewV4().String()
	newPswd := azgraphrbac.PasswordCredential{
		Value:   &secret,
		EndDate: &date.Time{Time: time.Now().AddDate(1, 0, 0)},
	}
	passwords = append(passwords, newPswd)

	_, err = am.ac.Patch(ctx, aroTeamSharedID, azgraphrbac.ApplicationUpdateParameters{
		AvailableToOtherTenants: to.BoolPtr(false),
		PasswordCredentials:     &passwords,
	})
	if err != nil {
		return err
	}

	// TODO: Rotate secrets
	// TODO: Update secret in CI cluster with new value
	// TODO: make 2 secret rotation cycle so we would not lock ourselfs out

	return nil
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}

}

func run() error {
	ctx := context.Background()
	am, err := newAADManager(ctx, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		return err
	}
	if err := am.rotateSecret(ctx); err != nil {
		panic(err)
	}
	return nil
}
