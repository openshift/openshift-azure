#!/bin/bash -e

cleanup() {
    set +e

    generate_artifacts

    delete real
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

export RUNNING_UNDER_TEST=false
export TEST_IN_PRODUCTION=true

hack/e2e.sh
