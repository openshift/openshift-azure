#!/bin/bash -e

cleanup() {
    set +e

    if [[ -n "$ARTIFACTS" ]]; then
        exec &>"$ARTIFACTS/cleanup"
    fi

    make artifacts

    if [[ -n "$NO_DELETE" ]]; then
        return
    fi
    make delete
    az group delete -g "$RESOURCEGROUP" --yes --no-wait
}
trap cleanup EXIT

. hack/tests/ci-operator-prepare.sh

export RUNNING_UNDER_TEST=false
export TEST_IN_PRODUCTION=true

make create

hack/e2e.sh
