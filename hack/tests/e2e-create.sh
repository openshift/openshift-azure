#!/bin/bash -e

cleanup() {
    set +e

    if [[ -n "$ARTIFACTS" ]]; then
        exec &>"$ARTIFACTS/cleanup"
    fi

    stop_monitoring
    make artifacts

    if [[ -n "$NO_DELETE" ]]; then
        return
    fi
    make delete
    az group delete -g "$RESOURCEGROUP" --yes --no-wait
}
trap cleanup EXIT

. hack/tests/ci-operator-prepare.sh

start_monitoring
set_build_images

make create e2e
