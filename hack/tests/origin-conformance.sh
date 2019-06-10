#!/bin/bash -ex

REAL="`realpath $0`"
HERE="`dirname ${REAL}`"

PATH=`go env GOPATH`/bin::$PATH
TEST_SUITE="${TEST_SUITE:-openshift/conformance/parallel/minimal}"

cleanup() {
  set +e

  generate_artifacts

  delete
}

fetch_origin() {
  local branch="${1:-release-3.11}"
  local remote="https://github.com/openshift/origin.git"

  test -d /tmp/origin || git clone --depth=1 --branch=${branch} $remote /tmp/origin
}

install_binaries() {
  # install ginkgo
  go get -v github.com/onsi/ginkgo/ginkgo

  # build extended.tests and/or openshift-tests (regular user can't write to /usr/local)
  make -C /tmp/origin build-extended-test && mv -v /tmp/origin/_output/local/bin/linux/`go env GOARCH`/* `go env GOPATH`/bin/

  # install kubectl if not available
  which kubectl || ( curl -s -L -o `go env GOPATH`/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/`go env GOARCH`/kubectl && chmod -v +x `go env GOPATH`/bin/kubectl )
}

function run-tests() {
  if which openshift-tests && [[ -n "${TEST_SUITE}" ]]; then
    openshift-tests run "${TEST_SUITE}" \
      --provider "${TEST_PROVIDER-}" \
      -o /tmp/artifacts/e2e.log \
      --junit-dir /tmp/artifacts/junit
    exit 0
  fi
  if ! which extended.test; then
    echo "must provide TEST_SUITE variable"
    exit 1
  fi
  # TODO: remove after this point once we move to 4.x
  ginkgo -v -noColor \
    -nodes="${TEST_NODES:-30}" \
    `which extended.test` -- \
    -ginkgo.focus="${TEST_SUITE}" \
    -e2e-output-dir /tmp/artifacts \
    -report-dir /tmp/artifacts/junit \
    -test.timeout=2h \
    -provider "${TEST_PROVIDER-}" \
    ${PROVIDER_ARGS-} || rc=$?

  exit ${rc:-0}
}

trap cleanup EXIT

. ${HERE}/ci-prepare.sh

start_monitoring
set_build_images

make create

if [ -n "$OS_GIT_MAJOR" -a -n "$OS_GIT_MINOR" ]; then
  BRANCH="release-${OS_GIT_MAJOR}.${OS_GIT_MINOR}"
fi

fetch_origin "$BRANCH"

install_binaries

run-tests
