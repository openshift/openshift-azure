package vaultmgmt

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE VaultMgmtClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type VaultMgmtClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, vaultName string, parameters mgmtkeyvault.VaultCreateOrUpdateParameters) (result mgmtkeyvault.Vault, err error)
}

type vaultsMgmtClient struct {
	mgmtkeyvault.VaultsClient
}

var _ VaultMgmtClient = &vaultsMgmtClient{}

// NewVaultMgmtClient get a new client for management actions
func NewVaultMgmtClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VaultMgmtClient {
	client := mgmtkeyvault.NewVaultsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "keyvault.VaultsClient", &client.Client, authorizer)

	return &vaultsMgmtClient{
		VaultsClient: client,
	}
}
