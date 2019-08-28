#!/bin/bash -e

cleanup() {
    set +e

    generate_artifacts

    delete
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

check_skip_ci

start_monitoring
set_build_images

make create

hack/e2e.sh
