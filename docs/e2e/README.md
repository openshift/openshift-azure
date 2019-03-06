#### Running e2e tests locally with go test

If you are developing tests locally, you can use the native `go test` to invoke your ginkgo test specs.

Example: Run the end user e2e tests
```
source ./env
export FOCUS="\[CustomerAdmin\]|\[EndUser\]\[Fake\]"
./hack/e2e.sh
```

As a shorthand to the above, you could simply run `make e2e`

Example: Run the scale up/down e2e tests against the fake rp
```
source ./env
export FOCUS="\[ScaleUpDown\]\[Fake\]"
export TIMEOUT=30m
export EXTRA_FLAGS="-scaleUpManifest=test/manifests/normal/scaleup.yaml -scaleDownManifest=test/manifests/normal/scaledown.yaml"
./hack/e2e.sh
```

#### Running e2e tests with the container image

To run the e2e tests in a container use the [e2e](https://quay.io/repository/openshift-on-azure/e2e-tests) container
image

```
docker pull quay.io/openshift-on-azure/e2e-tests:latest
```

Example: Run the end user e2e tests in a container
```
export FOCUS="\[CustomerAdmin\]|\[EndUser\]\[Fake\]"
export TIMEOUT=20m
source ./env

docker run \
--rm \
-it \
--volume $PWD/_data:/_data \
--volume $PWD/artifacts:/artifacts \
--volume $PWD/secrets:/secrets \
--volume $HOME/.kube/config:/root/.kube/config \
quay.io/openshift-on-azure/e2e-tests \
-test.v \
-test.timeout=${TIMEOUT} \
-ginkgo.noColor \
-ginkgo.v \
-ginkgo.focus=${FOCUS:-} \
-artifact-dir=/artifacts
```

Example: Run the scale up/down e2e tests against the fake rp in a container
```
export FOCUS="\[ScaleUpDown\]\[Fake\]"
export TIMEOUT=30m
source ./env

docker run \
--rm \
-it \
--env AZURE_CLIENT_ID=${AZURE_CLIENT_ID} \
--env AZURE_CLIENT_SECRET=${AZURE_CLIENT_SECRET} \
--env AZURE_SUBSCRIPTION_ID=${AZURE_SUBSCRIPTION_ID} \
--env AZURE_TENANT_ID=${AZURE_TENANT_ID} \
--env RESOURCEGROUP=${RESOURCEGROUP} \
--volume $PWD/_data:/_data \
--volume $PWD/artifacts:/artifacts \
--volume $PWD/secrets:/secrets \
--volume $PWD/test/manifests:/manifests \
--volume $HOME/.kube/config:/root/.kube/config \
quay.io/openshift-on-azure/e2e-tests \
-test.v \
-test.timeout=${TIMEOUT} \
-ginkgo.noColor \
-ginkgo.v \
-ginkgo.focus=${FOCUS:-} \
-artifact-dir=/artifacts \
-scaleUpManifest=/manifests/normal/scaleup.yaml \
-scaleDownManifest=/manifests/normal/scaledown.yaml
```
