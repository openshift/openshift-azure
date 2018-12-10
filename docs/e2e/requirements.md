#### Supported test focuses

The following are ginkgo test focuses supported by openshift on azure e2e tests

* `[EveryPR]` - These tests should be run on every PR
* `[LongRunning]` - These tests are long running and should be run periodically
* `[Real]` - These tests should be run against the real RP
* `[Fake]` - These tests should be run against the fake RP

All tests require the specification of either a [Real] or [Fake] focus. 

#### Customizing e2e test scope with focuses

You can specify a test focus by setting the `FOCUS` environment variable. For example
to run all tests tagged with [EveryPR] against the Real RP you would specify a focus as

```
export FOCUS=\[EveryPR\]\[Real\]
```

Note: even if this document refers to focuses such as [Real], you should always escape them
before passing them to `FOCUS`.

#### Real RP environment requirements

The following are required for running all e2e tests against the real RP

| Artifact Kind | Name | Notes |
|--- | --- | --- |
| `Ginkgo Focus` | `[Real]` | Allows the test to configure its `OpenshiftManagedCluster` client against the real RP |
| `Environment Variable` | `AZURE_CLIENT_ID` | Required for authentication against the real RP |
| `Environment Variable` | `AZURE_CLIENT_SECRET` | Required for authentication against the real RP |
| `Environment Variable` | `AZURE_TENANT_ID` | Required for authentication against the real RP |
| `Environment Variable` | `AZURE_SUBSCRIPTION_ID` | The subscription id for an existing cluster |
| `Environment Variable` | `AZURE_REGION` | The region for an existing cluster |
| `Environment Variable` | `RESOURCEGROUP` | The resource group for an existing cluster |

#### Fake RP environment requirements

The following are required for running all e2e tests against the fake RP

| Artifact Kind | Name | Notes |
| --- | --- | --- |
| `Ginkgo Focus` | `[Fake]` | Allows the test to configure its `OpenshiftManagedCluster` client against the fake RP |
| `Environment Variable` | `AZURE_CLIENT_ID` | Required for authentication against the RP |
| `Environment Variable` | `AZURE_CLIENT_SECRET` | Required for authentication against the RP |
| `Environment Variable` | `AZURE_TENANT_ID` | Required for authentication against the RP |
| `Environment Variable` | `AZURE_SUBSCRIPTION_ID` | The subscription id for an existing cluster |
| `Environment Variable` | `RESOURCEGROUP` | The resource group for an existing cluster |
| `File` | `_data/containerservice.yaml` | Allows the test to retrieve credentials as well as to decide if it is a `create` vs `update` |
| `File` | `_data/manifest.yaml` | The external cluster manifest (required for key rotation and reentrant updates tests) |
| `File` | `secrets/logging-int.cert` | Geneva logging client certificate |
| `File` | `secrets/logging-int.key` | Geneva logging client key |
| `File` | `secrets/metrics-int.cert` | Geneva metrics client certificate |
| `File` | `secrets/metrics-int.key` |  Geneva metrics client key |
| `File` | `secrets/.dockerconfigjson` |  Docker config allowing to pull geneva images |
| `File` | `test/manifests/normal/scaleup.yaml` |  Scale-up manifest (required for scale up/down tests) |
| `File` | `test/manifests/normal/scaledown.yaml` |  Scale-down manifest (required for scale up/down tests) |

#### Special considerations

##### Scale up/down
The scale up/down e2e test accepts two flags for customizing its functionality

* `scaleUpManifest`: Path to the scale up manifest (default: `test/manifests/normal/scaleup.yaml`)
* `scaleDownManifest`: Path to the scale down manifest (default: `test/manifests/normal/scaleup.yaml`)

Partial updates are supported. These flags are required for any test focus which includes the scale up/down test.

##### Key Rotation
The key rotation up/down e2e test accepts two flags for customizing its functionality

* `manifest`: Path to a customer manifest specifying a cluster for which keys should be rotated (default: `_data/manifest.yaml`)
* `configBlob`: Path to an internal config blob for the cluster in question (default: `_data/containerservice.yaml`)

These flags are required for any test focus which includes the key rotation test.

##### Reentrant Updates
The reentrant updates e2e test accepts one flag for customizing its functionality

* `manifest`: Path to a customer manifest specifying a cluster for which reentrant updates should be tested (default: `_data/manifest.yaml`)

These flags are required for any test focus which includes the reentrant updates test.

##### Etcd Recovery
The etcd recovery e2e test depends on the following file for its functionality. This is currently not configurable

* `_data/containerservice.yaml`: Path to an internal config blob for the cluster in question
