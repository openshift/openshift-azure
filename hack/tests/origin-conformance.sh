#!/bin/bash -e

PATH="$(go env GOPATH)/bin":$PATH
TEST_SUITE="${TEST_SUITE:-openshift/conformance/parallel/minimal}"

cleanup() {
  set +e

  generate_artifacts

  delete
}

fetch_origin() {
  local branch="$1"
  local remote="https://github.com/openshift/origin.git"

  test -d /tmp/origin || git clone --depth=1 --branch=${branch} $remote /tmp/origin
}

build_extended_test() {
  # build extended.tests (regular user can't write to /usr/local)
  # TODO: This will need updating once we move to 4.x
  mkdir -pv $(go env GOPATH)/bin
  make -C /tmp/origin build-extended-test
  mv -v /tmp/origin/_output/local/bin/linux/$(go env GOARCH)/* $(go env GOPATH)/bin/
}

function run_tests() {
  # TODO: This will need updating once we move to 4.x
  ginkgo -v -noColor \
    -nodes="${TEST_NODES:-30}" \
    $(which extended.test) -- \
    -ginkgo.focus="${TEST_SUITE}" \
    -e2e-output-dir /tmp/artifacts \
    -report-dir /tmp/artifacts/junit \
    -test.timeout=2h \
    -provider "${TEST_PROVIDER-}" \
    ${PROVIDER_ARGS-} || rc=$?

  exit ${rc:-0}
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

start_monitoring
set_build_images

make create

# TODO: This will need updating once we move to 4.x
fetch_origin "release-3.11"

build_extended_test

export KUBECONFIG=$PWD/_data/_out/admin.kubeconfig

run_tests
