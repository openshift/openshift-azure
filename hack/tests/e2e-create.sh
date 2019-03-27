#!/bin/bash -ex

cleanup() {
    set +e

    stop_monitoring
    make artifacts
    make delete
}
trap cleanup EXIT

. hack/tests/ci-operator-prepare.sh

start_monitoring
set_build_images

make create e2e
