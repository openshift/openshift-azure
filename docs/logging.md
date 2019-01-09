# OpenShift on Azure logging

#### [MQL reference](https://genevamondocs.azurewebsites.net/diagnostics_apps/dgrep/concepts/querylanguage/mql.html)

#### Example MQL queries

##### View entire journal for `master-000000`

```
select PreciseTimeStamp, _HOSTNAME, _COMM, _PID, MESSAGE as MSG
where _HOSTNAME == "master-000000"
orderby PreciseTimeStamp
```

##### View `kubelet` logs on `master-000000`

```
select PreciseTimeStamp, _HOSTNAME, _COMM, _PID, MESSAGE as MSG
where _HOSTNAME == "master-000000" and _COMM == "hyperkube"
orderby PreciseTimeStamp
```

##### View logs for the `api` container of pod `master-api-master-000000` in namespace `kube-system`

```
select PreciseTimeStamp, CONTAINER_NAME, MESSAGE as MSG
where CONTAINER_NAME != null and CONTAINER_NAME.startswith("k8s_api_master-api-master-000000_kube-system_")
orderby PreciseTimeStamp
```
