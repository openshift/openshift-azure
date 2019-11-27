# Query E2E test results

All E2E test result are being sent to `Application Insights"

`Azure Portal -> ResourceGroups -> insights -> ci -> Logs (Analytics)`


Example queries:

https://github.com/toddkitta/azure-content/blob/master/articles/application-insights/app-analytics-queries.md

https://docs.microsoft.com/en-us/azure/kusto/query/

## Tables:
`customMetrics` - individual test results. Selector `where customDimensions.type == "ginkgo"`
`customMetrics` - cluster creation time. Selector `where customDimensions.type == "cluster" and where customDimensions.metric == "creation"`

`traces` - all go tests logs
``
``

Data is being passed into `customMetrics` table in json format:
```
result := map[string]interface{}{
		"ComponentTexts": strings.Join(specSummary.ComponentTexts, " "),
		"RunTime":        specSummary.RunTime.String(),
		"FailureMessage": specSummary.Failure.Message,
		"Failed":         specSummary.Failed(),
		"Passed":         specSummary.Passed(),
		"Skipped":        specSummary.Skipped(),
	}
```

## Example queries

Results can be queries using Kusto queries below:

### Get all passed & failed test count in last 30 minutes

```
customMetrics
| extend componentText = extractjson('$.ComponentTexts', name)
| extend failed = extractjson('$.Failed', name)
| extend failureMessage = extractjson('$.FailureMessage', name)
| where failed = true
| where timestamp > ago(7d) 
| summarize count(componentText) by failed, componentText

```


### Get specific cluster upgrade availability status

```
// List clusters
requests
| where timestamp > ago(24h) 
| summarize count(name) by id
```

```
// Response time trend when upgrading cluster by cluster ID
requests
| summarize avgRequestDuration=avg(duration) by bin(timestamp, 30s),url, id
| where id =="<cluster_name>"
| render timechart
```

To remove y axe duration values higher than 0.5s:
```
requests
| summarize avgRequestDuration=iif(avg(duration) > real(500), real(500), avg(duration)) by bin(timestamp, 30s),url, id
| where id =="<cluster_name>"
| render timechart
```


### Get specific resourcegroup test logs

```
traces
| where customDimensions.type == "ginkgo"
| where customDimensions.resourcegroup == "<cluster_name>"
| where timestamp > ago(1d)
```
