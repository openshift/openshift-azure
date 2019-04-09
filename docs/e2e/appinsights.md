# Query E2E test results

All E2E test result are being sent to `Application Insights"

`Azure Portal -> ResourceGroups -> fakerp-upgrades-insights -> osa-fakeRP-upgrades -> Analytics`

Data is being passed into `customEvents` table in json format:
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
customEvents
| extend componentText = extractjson('$.ComponentTexts', name)
| extend failed = extractjson('$.Failed', name)
| extend failureMessage = extractjson('$.FailureMessage', name)
| where timestamp > ago(30m) 
| summarize count(componentText) by failed, componentText
| render barchart
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
| where id =="e2e-upgrade-asalkeld-1386-phmfxi"
| render timechart
```
