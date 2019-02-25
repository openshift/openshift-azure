package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-09-01/insights"
)

const (
	ActivityLogSelectParams       = "status,subscriptionId,resourceId,eventName,operationName,httpRequest"
	ActivityLogByResourceIdFilter = "eventTimestamp ge '%s' and eventTimestamp le '%s' and resourceUri eq %s"
)

type ActivityLogStatus string

var (
	ActivityStarted    ActivityLogStatus = "Started"
	ActivityInProgress ActivityLogStatus = "In progress"
	ActivitySucceeded  ActivityLogStatus = "Succeeded"
	ActivityFailed     ActivityLogStatus = "Failed"
	ActivityResolved   ActivityLogStatus = "Resolved"
)

// OSAResourceGroup returns the name of the resource group holding the OSA
// cluster resources
func (cli *Client) OSAResourceGroup(ctx context.Context, resourcegroup, name, location string) (string, error) {
	appName := strings.Join([]string{"OS", resourcegroup, name, location}, "_")

	app, err := cli.Applications.Get(ctx, appName, appName)
	if err != nil {
		return "", err
	}
	if app.ApplicationProperties == nil {
		return "", fmt.Errorf("managed application %q not found", appName)
	}

	// can't use azure.ParseResourceID here because rgid is of the short form
	// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}
	rgid := *app.ApplicationProperties.ManagedResourceGroupID
	return rgid[strings.LastIndexByte(rgid, '/')+1:], nil
}

// ActivityLogsByResourceId returns the activity logs for a given resource id within a time interval
func (cli *Client) ActivityLogsByResourceId(ctx context.Context, resourceId string, start, end time.Time) ([]insights.EventData, error) {
	filter := fmt.Sprintf(ActivityLogByResourceIdFilter, start.Format(time.RFC3339), end.Format(time.RFC3339), resourceId)

	pages, err := cli.ActivityLogs.List(context.Background(), filter, ActivityLogSelectParams)
	if err != nil {
		return nil, err
	}
	var logs []insights.EventData
	for pages.NotDone() {
		logs = append(logs, pages.Values()...)
		err = pages.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return logs, nil
}

// ActivityLogsByResourceIdAndStatus returns the activity logs of a given status for a given resource id within a time interval
func (cli *Client) ActivityLogsByResourceIdAndStatus(ctx context.Context, resourceId string, start, end time.Time, status ActivityLogStatus) ([]insights.EventData, error) {
	logs, err := cli.ActivityLogsByResourceId(ctx, resourceId, start, end)
	if err != nil {
		return nil, err
	}
	var filtered []insights.EventData
	for _, log := range logs {
		if ActivityLogStatus(*log.Status.Value) == ActivitySucceeded {
			filtered = append(filtered, log)
		}
	}
	return filtered, nil
}
