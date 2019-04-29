package dns

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
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
func NewZonesClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) ZonesClient {
	client := dns.NewZonesClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "dns.ZonesClient", &client.Client, authorizer)

	return &zonesClient{
		ZonesClient: client,
	}
}
