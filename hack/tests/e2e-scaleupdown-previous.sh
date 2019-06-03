#!/bin/bash -e

if [[ $# -ne 1 ]]; then
    echo "usage: $0 source_version"
    exit 1
fi

cleanup() {
    set +e

    generate_artifacts

    if [[ -n "$T" ]]; then
        rm -rf "$T"
    fi

    delete
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

T="$(mktemp -d)"
start_monitoring $T/src/github.com/openshift/openshift-azure/_data/containerservice.yaml

git clone -q -b "$1" https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure
ln -sf "$PWD/secrets" "$T/src/github.com/openshift/openshift-azure"
(
    export AZURE_MASTER_CLIENT_ID=$AZURE_LEGACY_MASTER_CLIENT_ID
    export AZURE_MASTER_CLIENT_SECRET=$AZURE_LEGACY_MASTER_CLIENT_SECRET
    export AZURE_WORKER_CLIENT_ID=$AZURE_LEGACY_WORKER_CLIENT_ID
    export AZURE_WORKER_CLIENT_SECRET=$AZURE_LEGACY_WORKER_CLIENT_SECRET
    cd "$T/src/github.com/openshift/openshift-azure"
    GOPATH="$T" make create
)

cp -a "$T/src/github.com/openshift/openshift-azure/_data" .

set_build_images

FOCUS="\[ScaleUpDown\]\[Fake\]" ./hack/e2e.sh
