package azureclient

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-09-01/insights"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"
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
	setupClient(ctx, log, "insights.ActivityLogsClient", &client.Client, authorizer)

	return &activityLogsClient{
		ActivityLogsClient: client,
	}
}
