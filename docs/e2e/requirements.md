#### Supported test focuses

The following are ginkgo test focuses supported by openshift on azure e2e tests

* `[EveryPR]` - These tests should be run on every PR
* `[LongRunning]` - These tests are long running and should be run periodically
* `[CustomerAdmin]` - These test run with `make e2e` and focus on Customer Admin capabilities
* `[EndUser]` - These test run with `make e2e` and focus on End User capabilities
* `[Apps]` - These test creates and validates a test application in an OpenShift cluster _(this test gets ran with `[EndUser]` focus as well)_

#### Customizing e2e test scope with focuses

You can specify a test focus by setting the `FOCUS` environment variable. For example
to run all tests tagged with [EveryPR] you would specify the focus as

```bash
export FOCUS=\[EveryPR\]
```

or if you want to tests two focuses

```bash
export FOCUS="\[CustomerAdmin\]|\[EndUser\]"
 ```

**Note:** Even if this document refers to focuses such as `[EveryPR]`, you should always escape them
before passing them to `FOCUS`.

#### Environment requirements

The following are required for running all e2e tests against the fake RP.  To generate these files you can use `make secrets` at the root of this repository.  See [environment file](../../README.md#prerequisites) to see how to retrieve the environment variables.

| Artifact Kind | Name | Notes |
| --- | --- | --- |
| `Environment Variable` | `AZURE_CLIENT_ID` | Required for authentication against the RP |
| `Environment Variable` | `AZURE_CLIENT_SECRET` | Required for authentication against the RP |
| `Environment Variable` | `AZURE_TENANT_ID` | Required for authentication against the RP |
| `Environment Variable` | `AZURE_SUBSCRIPTION_ID` | The subscription id for an existing cluster |
| `Environment Variable` | `RESOURCEGROUP` | The resource group for an existing cluster |
| `File` | `_data/containerservice.yaml` | Required for openshift client setup |
| `File` | `secrets/logging-int.cert` | Geneva logging client certificate |
| `File` | `secrets/logging-int.key` | Geneva logging client key |
| `File` | `secrets/metrics-int.cert` | Geneva metrics client certificate |
| `File` | `secrets/metrics-int.key` |  Geneva metrics client key |
| `File` | `secrets/acr-docker-pull-secret` |  Docker config allowing to pull geneva images |
| `File` | `secrets/rh-docker-pull-secret"` |  Docker config allowing to pull red hat secured registry images |
