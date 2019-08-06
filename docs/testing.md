# OpenShift on Azure testing

OpenShift on Azure project is using a few testing methods. 
Some of the test code base is being reused from other projects.

PR commands to execute tests:
```
/test e2e - run native e2e tests
/test unit - run unit tests
```

Optional test commands:
```
/test upgrade - run upgrade in-place cluster tests
/test conformance - run origin conformance tests
/test etcdbackuprecovery - run etcd backup and restore test
/test keyrotation - run key rotation test
/test e2e-no-test - no tests
/test prod - run prod tests (broken)
/test scaleupdown - run cluster scale-up-down test
/test upgrade-v1.2 - run upgrade release version v1.2 to PR code test
/test upgrade-v2.0 - run upgrade release version v2.0 to PR code test
/test upgrade-v2.1 - run upgrade release version v2.1 to PR code test
```

We have some optinal jobs, which are artifacts of prow-gen.
Those can be ignored and `/skip`ed. In example:
```
/test e2e-upgrade-v1.2
/test e2e-upgrade-v2.0
/test e2e-upgrade-v2.1
```
Upstream issue: https://github.com/openshift/ci-operator-prowgen/issues/79 

## Unit test

Packages under `pkg` directory contain their individual unit tests.

To run all unit tests locally, execute: `make unit`

```makefile
unit: generate testinsights
  go test ./... -coverprofile=coverage.out -covermode=atomic -json | ./testinsights
```

or

```go 
go test ./...
```

To run a single package's tests:
`go test ./pkg/util/tls`

To run a subset of tests in a package:
`go test -run TestFoo ./pkg/util/tls`

## OpenShift E2E tests

The project has its own end-to-end testing test suite. You can find it under:
`test/e2e/`. It uses [ginko](https://github.com/onsi/ginkgo) and [gomega](https://github.com/onsi/gomega).

To run these tests you will need to have OpenShift cluster running.

The cluster running that you are trying to test **must** have the required files and environment variables described in [e2e requirements](e2e/requirements.md).

To execute e2e tests **locally**, execute:
`make e2e`   

```Makefile
e2e:
  FOCUS="\[CustomerAdmin\]|\[EndUser\]" TIMEOUT=60m ./hack/e2e.sh
```

or
```golang
go test -tags e2e ./test/e2e
```

See [e2e requirements](e2e/requirements.md) for more information on the expected inputs to our e2e tests.

See [running e2e](e2e/README.md) for more information on how to run the e2e tests with the container image.

## OpenShift Origin Conformance tests

For the generic OpenShift functionality we use the [OpenShift Origin](https://github.com/openshift/origin)
conformance testing suite. All test configuration for those tests can be
found in [OpenShift Release repository](https://github.com/openshift/release/)

### Conformance test development

If you want just to run conformance test **locally**, you can use docker image to do so.

```
# run and attach to test the container
docker run -v $(pwd)/_data:/tmp/_data  -it openshift/origin-tests:v3.11 sh
# export kubeconfig
export KUBECONFIG=/tmp/_data/_out/admin.kubeconfig
# filter tests
# export TEST_FOCUS=
# export TEST_SKIP=".*((The HAProxy router should set Forwarded headers appropriately)).*"
# run tests
/usr/libexec/origin/ginkgo /usr/libexec/origin/extended.test -v -noColor -nodes=30 extended.test \
-- -ginkgo.focus="Suite:openshift/conformance/parallel" -e2e-output-dir /tmp/ \
-report-dir /tmp/_data/junit -test.timeout=2h  -ginkgo.focus="${TEST_FOCUS}" \
-ginkgo.skip="${TEST_SKIP}"
```

Example:
```
/usr/libexec/origin/ginkgo /usr/libexec/origin/extended.test -v -noColor \
-nodes=30 extended.test -- -ginkgo.focus="Suite:openshift/conformance/parallel" \
 -test.timeout=2h  -ginkgo.focus=".*should report failed soon after an annotated objects has failed*."
```

If you need to update `conformance` tests:
```
# fork github.com/openshift/origin repository
mkdir -p $GOPATH/src/github.com/openshift
git clone https://github.com/<gh_username>/origin $GOPATH/src/github.com/openshift/origin
cd $GOPATH/src/github.com/openshift/origin
# build extended test
make build-extended-test
# export kubeconfig
export KUBECONFIG=$(pwd)/_data/_out/admin.kubeconfig
# run test via wrapper
TEST_ONLY=1 test/extended/core.sh -test.timeout=2h -ginkgo.focus="${TEST_FOCUS}" \
-report-dir /tmp/_data/junit
```
