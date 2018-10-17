package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest"
)

// ZonesClient is a minimal interface for azure ZonesClient
type ZonesClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, zoneName string, parameters dns.Zone, ifMatch string, ifNoneMatch string) (dns.Zone, error)
	Delete(ctx context.Context, resourceGroupName string, zoneName string, ifMatch string) (dns.ZonesDeleteFuture, error)
	Client
}

type zonesClient struct {
	dns.ZonesClient
}

var _ ZonesClient = &zonesClient{}

// NewZonesClient creates a new NewZonesClient
func NewZonesClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) ZonesClient {
	client := dns.NewZonesClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)

	return &zonesClient{
		ZonesClient: client,
	}
}

func (c *zonesClient) Client() autorest.Client {
	return c.ZonesClient.Client
}

// RecordSetsClient is a minimal interface for azure RecordSetsClient
type RecordSetsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType, parameters dns.RecordSet, ifMatch string, ifNoneMatch string) (dns.RecordSet, error)
	Delete(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType, ifMatch string) (autorest.Response, error)
	Client
}

type recordSetsClient struct {
	dns.RecordSetsClient
}

var _ RecordSetsClient = &recordSetsClient{}

// NewRecordSetsClient creates a new NewRecordSetsClient
func NewRecordSetsClient(subscriptionID string, authorizer autorest.Authorizer, languages []string) RecordSetsClient {
	client := dns.NewRecordSetsClient(subscriptionID)
	client.Authorizer = authorizer
	client.RequestInspector = addAcceptLanguages(languages)

	return &recordSetsClient{
		RecordSetsClient: client,
	}
}

func (c *recordSetsClient) Client() autorest.Client {
	return c.RecordSetsClient.Client
}
