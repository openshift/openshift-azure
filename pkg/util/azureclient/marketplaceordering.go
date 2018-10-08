package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/marketplaceordering/mgmt/2015-06-01/marketplaceordering"
	"github.com/Azure/go-autorest/autorest"
)

// MarketPlaceAgreementsClient is a minimal interface for azure MarketPlaceAgreementsClient
type MarketPlaceAgreementsClient interface {
	Create(ctx context.Context, publisherID string, offerID string, planID string, parameters marketplaceordering.AgreementTerms) (result marketplaceordering.AgreementTerms, err error)
	Get(ctx context.Context, publisherID string, offerID string, planID string) (result marketplaceordering.AgreementTerms, err error)
}

type marketPlaceAgreementsClient struct {
	marketplaceordering.MarketplaceAgreementsClient
}

var _ MarketPlaceAgreementsClient = &marketPlaceAgreementsClient{}

// NewMarketPlaceAgreementsClient creates a new MarketPlaceAgreementsClient
func NewMarketPlaceAgreementsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) MarketPlaceAgreementsClient {
	client := marketplaceordering.NewMarketplaceAgreementsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)

	return &marketPlaceAgreementsClient{
		MarketplaceAgreementsClient: client,
	}
}
