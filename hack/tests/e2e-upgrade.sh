#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo "usage: $0 source_version"
    exit 1
fi

cleanup() {
    set +e

    stop_monitoring
    make artifacts
    make delete

    if [[ -n "$T" ]]; then
        rm -rf "$T"
    fi
}
trap cleanup EXIT

. hack/tests/ci-operator-prepare.sh

T="$(mktemp -d)"
start_monitoring $T/src/github.com/openshift/openshift-azure/_data/containerservice.yaml

git clone -b "$1" https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure
pushd "$T/src/github.com/openshift/openshift-azure"
setup_secrets
GOPATH="$T" make create
popd

cp -a "$T/src/github.com/openshift/openshift-azure/_data" .

set_build_images

ADMIN_MANIFEST=test/manifests/fakerp/admin-update.yaml make upgrade e2e
