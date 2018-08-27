### Centralised logging

#### Architecture

At cluster creation time, the ARM template creates a dedicated [Log
Analytics](https://azure.microsoft.com/en-us/services/log-analytics) resource in
the cluster resource group.  It is set up to use its default (minimum) retention
setting of 30 days of logs.

A `logbridge` process runs as a static pod on each VM.  It follows the systemd
journal and writes each journal entry to the Log Analytics resource via its REST
API.  The `logbridge` process regularly persists its latest journal cursor in
`/var/lib/logbridge/cursor` on the host.

The Docker daemon on master and infra nodes (but not on compute nodes) is
configured to log to the systemd journal.  This makes logs for most non-customer
containerised workload available via Log Analytics.

#### Usage

There are multiple consoles for querying stored logs.  An easy approach is to
use the Azure portal.  Navigate to the appropriate Log Analytics resource and
select `GENERAL` / `Logs (preview)`.

Some example queries are shown below.  Consult the Log Analytics [Language
Reference](https://docs.loganalytics.io/docs/Language-Reference) for details on
the query language.

It takes the best part of an hour after cluster creation before logs start to
become visible in the Log Analytics resource.  Subsequently, log lines appear to
arrive with a latency of around five minutes.  Beware that it seems that log
lines do not necessarily become visible in Log Analytics in order.

All cluster logs are imported into the `osa_CL` table.

Log fields imported to Log Analytics should match those in the systemd journal.
With a couple of exceptions (`TimeGenerated`, `Message`), the Log Analytics
import process appends `_s` to fields of type string, and `_g` to fields of type
GUID.

The Log Analytics timestamp granularity is only at the millisecond level.  To
ensure that you view log lines in the correct order, add an `order by
__MONOTONIC_TIMESTAMP_d asc` clause in your query.

#### Example queries

##### View entire journal for `master-000000`

```
osa_CL
| where _HOSTNAME_s == "master-000000"
| order by __MONOTONIC_TIMESTAMP_d asc
| project TimeGenerated, _COMM_s, _PID_s, Message
```

##### View `kubelet` logs on `master-000000`

```
osa_CL
| where _HOSTNAME_s == "master-000000" and _COMM_s == "hyperkube"
| order by __MONOTONIC_TIMESTAMP_d asc
| project TimeGenerated, Message
```

##### View logs for the `api` container of pod `api-master-000000` in namespace `kube-system`

```
osa_CL
| where CONTAINER_NAME_s hasprefix "k8s_api_api-master-000000_kube-system"
| order by __MONOTONIC_TIMESTAMP_d asc
| project TimeGenerated, CONTAINER_ID_s, Message
```

##### Search for string "No API exists" in all cluster logs

```
osa_CL
| search "No API exists"
| order by __MONOTONIC_TIMESTAMP_d asc
| project TimeGenerated, _HOSTNAME_s, _COMM_s, _PID_s, CONTAINER_ID_s, Message
```
