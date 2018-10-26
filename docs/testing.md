# OpenShift on Azure testing

OpenShift on Azure project is using a few testing methods. 
Some of the test code base is being reused from other projects.

PR commands to execute tests:
```
/test e2e - run native e2e tests
/test unit - run unit tests
/test upgrade - run upgrade cluster tests
/test conformance - run origin conformance tests
```

## Unit test

Packages under `pkg` directory contain their individual unit tests.
To run all unit tests locally, execute: `make unit` or `go test ./....`
To run a single package's tests:
`go test ./pkg/tls`

To run a subset of tests in a package:
`go test -run TestFoo ./pkg/tls`

## OpenShift E2E tests

The project has its own end-to-end testing test suite. You can find it under:
`test/e2e/`. It uses [ginko](https://github.com/onsi/ginkgo) and [gomega](https://github.com/onsi/gomega).

To run these tests you will need to have OpenShift cluster running.
to execute e2e tests locally, execute: `make e2e` or `go test -tags e2e ./test/e2e`

## OpenShift Origin Conformance tests

For the generic OpenShift functionality we use the [OpenShift Origin](https://github.com/openshift/origin)
conformance testing suite. All test configuration for those tests can be
found in [OpenShift Release repository](https://github.com/openshift/release/)

### Conformance test development

If you want just to run conformance test locally, you can use docker image to do so.
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
