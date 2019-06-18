# Query

All data is being sent to `Application Insights"

`Azure Portal -> ResourceGroups -> fakerp-upgrades-insights -> osa-fakeRP-upgrades -> Analytics`


Example queries:

https://github.com/toddkitta/azure-content/blob/master/articles/application-insights/app-analytics-queries.md

https://docs.microsoft.com/en-us/azure/kusto/query/

## Tables:
`customMetrics` - cluster creation time. Selector:

```
customMetrics
| where customDimensions.type == "install"
| where customDimensions.version == "v4" 
```

## Example queries

Results can be queries using Kusto queries below:

TBC
