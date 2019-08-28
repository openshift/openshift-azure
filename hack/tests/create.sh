#!/bin/bash -e

cleanup() {
    set +e

    generate_artifacts

    delete
}

. hack/tests/ci-prepare.sh

check_skip_ci

start_monitoring
set_build_images

make create
