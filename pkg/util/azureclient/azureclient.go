package azureclient

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/marketplaceordering/mgmt/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	azmarketplaceordering "github.com/Azure/azure-sdk-for-go/services/marketplaceordering/mgmt/2015-06-01/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/openshift/openshift-azure/pkg/api"
)

// ClientWaitForCompletion base interface to return the client used in WaitForCompletionRef
type ClientWaitForCompletion interface {
	GetClient() autorest.Client
}

// DeploymentClient is minimal interface for azure DeploymentClient
type DeploymentClient interface {
	ClientWaitForCompletion
	CreateOrUpdate(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) (result resources.DeploymentsCreateOrUpdateFuture, err error)
}

// azDeploymentClient implements DeploymentClient.
type azDeploymentClient struct {
	client resources.DeploymentsClient
}

// AccountsClient is minimal interface for azure AccountsClient
type AccountsClient interface {
	// mirrored methods
	ListKeys(context context.Context, resourceGroup, accountName string) (storage.AccountListKeysResult, error)
	ListByResourceGroup(context context.Context, resourceGroup string) (storage.AccountListResult, error)
	// custom methods
	GetStorageAccount(ctx context.Context, resourceGroup, typeTag string) (map[string]string, error)
	GetStorageAccountKey(ctx context.Context, resourceGroup, accountName string) (string, error)
}

// azAccountsClient implements AccountsClient.
type azAccountsClient struct {
	client storage.AccountsClient
}

// StorageClient is minimal inferface for azure StorageClient
type StorageClient interface {
	// mirrored methods
	GetContainerReference(name string) *azstorage.Container
	GetBlobService() azstorage.BlobStorageClient
}

// azDeploymentClient implements DeploymentClient.
type azStorageClient struct {
	azs azstorage.Client
	bs  azstorage.BlobStorageClient
}

// MarketPlaceAgreementsClient is minimal interface for azure MarketPlaceAgreementsClient
type MarketPlaceAgreementsClient interface {
	Get(ctx context.Context, publisherID string, offerID string, planID string) (marketplaceordering.AgreementTerms, error)
	Create(ctx context.Context, publisherID string, offerID string, planID string, parameters marketplaceordering.AgreementTerms) (marketplaceordering.AgreementTerms, error)
}

// azMarketPlaceAgreementsClient implements MarketPlaceAgreementsClient.
type azMarketPlaceAgreementsClient struct {
	client azmarketplaceordering.MarketplaceAgreementsClient
}

// VirtualMachineScaleSetsClient is minimal interface for azure VirtualMachineScaleSetsClient
type VirtualMachineScaleSetsClient interface {
	ClientWaitForCompletion
	// mirrored methods
	Update(ctx context.Context, resourceGroupName string, VMScaleSetName string, parameters compute.VirtualMachineScaleSetUpdate) (compute.VirtualMachineScaleSetsUpdateFuture, error)
	UpdateInstances(ctx context.Context, resourceGroupName string, VMScaleSetName string, VMInstanceIDs compute.VirtualMachineScaleSetVMInstanceRequiredIDs) (compute.VirtualMachineScaleSetsUpdateInstancesFuture, error)
}

// azMarketPlaceAgreementsClient implements MarketPlaceAgreementsClient.
type azVirtualMachineScaleSetsClient struct {
	client compute.VirtualMachineScaleSetsClient
}

// VirtualMachineScaleSetVMsClient is minimal interface for azure VirtualMachineScaleSetVMsClient
type VirtualMachineScaleSetVMsClient interface {
	ClientWaitForCompletion
	// mirrored methods
	List(ctx context.Context, resourceGroupName string, virtualMachineScaleSetName string, filter string, selectParameter string, expand string) (compute.VirtualMachineScaleSetVMListResultPage, error)
	Delete(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsDeleteFuture, error)
	Deallocate(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsDeallocateFuture, error)
	Reimage(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsReimageFuture, error)
	Start(ctx context.Context, resourceGroupName string, VMScaleSetName string, instanceID string) (compute.VirtualMachineScaleSetVMsStartFuture, error)
}

// azMarketPlaceAgreementsClient implements MarketPlaceAgreementsClient.
type azVirtualMachineScaleSetVMsClient struct {
	client compute.VirtualMachineScaleSetVMsClient
}

func addAcceptLanguages(acceptLanguages []string) autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				return r, err
			}
			if acceptLanguages != nil {
				for _, language := range acceptLanguages {
					r.Header.Add("Accept-Language", language)
				}
			}
			return r, nil
		})
	}
}

func NewAuthorizerFromCtx(ctx context.Context) (autorest.Authorizer, error) {
	config := auth.NewClientCredentialsConfig(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
	return config.Authorizer()
}

func NewAuthorizer(clientID, clientSecret, tenantID, subscriptionID string) (autorest.Authorizer, error) {
	config := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	return config.Authorizer()
}
