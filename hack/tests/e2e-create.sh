#!/bin/bash -e

cleanup() {
    set +e

    generate_artifacts

    delete
}

trap cleanup EXIT

echo "Prepate CI"

cp - R /secrets $(pwd)/secrets

. hack/tests/ci-prepare.sh


echo "RESOURCEGROUP is $RESOURCEGROUP"

ls -la /
ls -la /secrets/

start_monitoring
set_build_images

make create

hack/e2e.sh
