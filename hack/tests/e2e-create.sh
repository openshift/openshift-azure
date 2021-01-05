#!/bin/bash -e

cleanup() {
    set +e

    generate_artifacts

    delete
}

trap cleanup EXIT

pwd

. hack/tests/ci-prepare.sh


start_monitoring
set_build_images

make create

hack/e2e.sh
