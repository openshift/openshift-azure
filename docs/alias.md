# ARO useful aliases

## Activity logs for the cluster
```
alias cluster-logs="az monitor activity-log list -g $RESOURCEGROUP --offset 7d --query "[].[eventTimestamp,submissionTimestamp,level,resourceId,eventName.value,operationName.value,status.value]" --max-events 400 -o tsv | column -t
```
