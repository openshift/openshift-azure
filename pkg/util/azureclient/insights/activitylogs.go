package insights

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE ActivityLogsClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-09-01/insights"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// ActivityLogsClient is a minimal interface for azure ActivityLogsClient
type ActivityLogsClient interface {
	List(ctx context.Context, filter string, selectParameter string) (result insights.EventDataCollectionPage, err error)
}

type activityLogsClient struct {
	insights.ActivityLogsClient
}

var _ ActivityLogsClient = &activityLogsClient{}

// NewActivityLogsClient creates a new ActivityLogsClient
func NewActivityLogsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) ActivityLogsClient {
	client := insights.NewActivityLogsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "insights.ActivityLogsClient", &client.Client, authorizer)

	return &activityLogsClient{
		ActivityLogsClient: client,
	}
}
