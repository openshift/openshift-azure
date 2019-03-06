package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest"
)

// ZonesClient is a minimal interface for azure ZonesClient
type ZonesClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, zoneName string, parameters dns.Zone, ifMatch string, ifNoneMatch string) (result dns.Zone, err error)
	ZonesClientAddons
}

type zonesClient struct {
	dns.ZonesClient
}

var _ ZonesClient = &zonesClient{}

// NewZonesClient creates a new ZonesClient
func NewZonesClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) ZonesClient {
	client := dns.NewZonesClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

	return &zonesClient{
		ZonesClient: client,
	}
}

// RecordSetsClient is a minimal interface for azure RecordSetsClient
type RecordSetsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType, parameters dns.RecordSet, ifMatch string, ifNoneMatch string) (result dns.RecordSet, err error)
	Delete(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType, ifMatch string) (result autorest.Response, err error)
	Get(ctx context.Context, resourceGroupName string, zoneName string, relativeRecordSetName string, recordType dns.RecordType) (result dns.RecordSet, err error)
}

type recordSetsClient struct {
	dns.RecordSetsClient
}

var _ RecordSetsClient = &recordSetsClient{}

// NewRecordSetsClient creates a new RecordSetsClient
func NewRecordSetsClient(ctx context.Context, subscriptionID string, authorizer autorest.Authorizer) RecordSetsClient {
	client := dns.NewRecordSetsClient(subscriptionID)
	setupClient(ctx, &client.Client, authorizer)

	return &recordSetsClient{
		RecordSetsClient: client,
	}
}
