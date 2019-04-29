package keyvault

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE KeyVaultClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type KeyVaultClient interface {
	GetSecret(ctx context.Context, vaultBaseURL string, secretName string, secretVersion string) (result keyvault.SecretBundle, err error)
	ImportCertificate(ctx context.Context, vaultBaseURL string, certificateName string, parameters keyvault.CertificateImportParameters) (result keyvault.CertificateBundle, err error)
}

type keyVaultsClient struct {
	keyvault.BaseClient
}

var _ KeyVaultClient = &keyVaultsClient{}

// NewKeyVaultClient gets a new client for accessing vault values.  Important:
// the authorizer supplied must have its resource set to
// "https://vault.azure.net"
func NewKeyVaultClient(ctx context.Context, log *logrus.Entry, authorizer autorest.Authorizer) KeyVaultClient {
	client := keyvault.New()
	azureclient.SetupClient(ctx, log, "keyvault.BaseClient", &client.Client, authorizer)

	return &keyVaultsClient{
		BaseClient: client,
	}
}
