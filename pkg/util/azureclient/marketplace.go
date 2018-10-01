package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/marketplaceordering/mgmt/marketplaceordering"
	"github.com/Azure/go-autorest/autorest"
	"github.com/openshift/openshift-azure/pkg/api"
)

// NewMarketPlaceAgreementsClient return MarketPlaceAgreementsClient implementation
func NewMarketPlaceAgreementsClient(subscriptionID string, authorizer autorest.Authorizer, pluginConfig api.PluginConfig) MarketPlaceAgreementsClient {
	client := marketplaceordering.NewMarketplaceAgreementsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(pluginConfig.AcceptLanguages)
	return &azMarketPlaceAgreementsClient{
		client: client,
	}
}

func (az azMarketPlaceAgreementsClient) Get(ctx context.Context, publisherID string, offerID string, planID string) (result marketplaceordering.AgreementTerms, err error) {
	return az.client.Get(ctx, publisherID, offerID, planID)
}

func (az azMarketPlaceAgreementsClient) Create(ctx context.Context, publisherID string, offerID string, planID string, parameters marketplaceordering.AgreementTerms) (result marketplaceordering.AgreementTerms, err error) {
	return az.client.Create(ctx, publisherID, offerID, planID, parameters)
}
