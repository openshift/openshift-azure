package azureclient

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/marketplaceordering/mgmt/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	azmarketplaceordering "github.com/Azure/azure-sdk-for-go/services/marketplaceordering/mgmt/2015-06-01/marketplaceordering"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/openshift/openshift-azure/pkg/api"
)

// AzureClients holds all the used clients used by openshift-azure
type AzureClients struct {
	Accounts                  storage.AccountsClient
	MarketPlaceAgreements     azmarketplaceordering.MarketplaceAgreementsClient
	Deployments               resources.DeploymentsClient
	VirtualMachineScaleSets   compute.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetVMs compute.VirtualMachineScaleSetVMsClient
}

// NewAzureClients create all the clients we need (not to be called by the sync pod)
func NewAzureClients(ctx context.Context, cs *api.OpenShiftManagedCluster, pluginConfig api.PluginConfig) (*AzureClients, error) {
	config := auth.NewClientCredentialsConfig(ctx.Value(api.ContextKeyClientID).(string), ctx.Value(api.ContextKeyClientSecret).(string), ctx.Value(api.ContextKeyTenantID).(string))
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, err
	}

	clients := &AzureClients{}

	clients.Deployments = resources.NewDeploymentsClient(cs.Properties.AzProfile.SubscriptionID)
	clients.Deployments.Authorizer = authorizer
	clients.Deployments.Client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)

	clients.MarketPlaceAgreements = marketplaceordering.NewMarketplaceAgreementsClient(cs.Properties.AzProfile.SubscriptionID)
	clients.MarketPlaceAgreements.Authorizer = authorizer
	clients.MarketPlaceAgreements.Client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)

	clients.VirtualMachineScaleSets = compute.NewVirtualMachineScaleSetsClient(cs.Properties.AzProfile.SubscriptionID)
	clients.VirtualMachineScaleSets.Authorizer = authorizer
	clients.VirtualMachineScaleSets.Client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)

	clients.VirtualMachineScaleSetVMs = compute.NewVirtualMachineScaleSetVMsClient(cs.Properties.AzProfile.SubscriptionID)
	clients.VirtualMachineScaleSetVMs.Authorizer = authorizer
	clients.VirtualMachineScaleSetVMs.Client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)

	clients.Accounts = storage.NewAccountsClient(cs.Properties.AzProfile.SubscriptionID)
	clients.Accounts.Authorizer = authorizer
	clients.Accounts.Client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)

	return clients, nil
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
