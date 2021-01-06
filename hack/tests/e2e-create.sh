#!/bin/bash -e

cleanup() {
    set +e

    generate_artifacts

    delete
}

trap cleanup EXIT

echo "Prepate CI"

. hack/tests/ci-prepare.sh

echo "RESOURCEGROUP is $RESOURCEGROUP"

start_monitoring
set_build_images

make create

hack/e2e.sh
