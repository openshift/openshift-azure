#!/bin/bash -e

cleanup() {
    set +e

    generate_artifacts

    delete
}

pwd

. hack/tests/ci-prepare.sh

start_monitoring
set_build_images

make create
