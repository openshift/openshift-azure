#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo "usage: $0 source_version"
    exit 1
fi

cleanup() {
    set +e

    if [[ -n "$ARTIFACTS" ]]; then
        exec &>"$ARTIFACTS/cleanup"
    fi

    stop_monitoring
    make artifacts

    if [[ -n "$T" ]]; then
        rm -rf "$T"
    fi

    if [[ -n "$NO_DELETE" ]]; then
        return
    fi
    make delete
    az group delete -g "$RESOURCEGROUP" --yes --no-wait
}
trap cleanup EXIT

. hack/tests/ci-operator-prepare.sh

T="$(mktemp -d)"
start_monitoring $T/src/github.com/openshift/openshift-azure/_data/containerservice.yaml

git clone -b "$1" https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure
ln -sf "$PWD/secrets" "$T/src/github.com/openshift/openshift-azure"
(
    set +x
    export AZURE_MASTER_CLIENT_ID=$AZURE_LEGACY_MASTER_CLIENT_ID
    export AZURE_MASTER_CLIENT_SECRET=$AZURE_LEGACY_MASTER_CLIENT_SECRET
    export AZURE_WORKER_CLIENT_ID=$AZURE_LEGACY_WORKER_CLIENT_ID
    export AZURE_WORKER_CLIENT_SECRET=$AZURE_LEGACY_WORKER_CLIENT_SECRET
    set -x
    cd "$T/src/github.com/openshift/openshift-azure"
    GOPATH="$T" make create
)

cp -a "$T/src/github.com/openshift/openshift-azure/_data" .

set_build_images

make e2e-scaleupdown
