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
    cd "$T/src/github.com/openshift/openshift-azure"
    GOPATH="$T" make create
)

cp -a "$T/src/github.com/openshift/openshift-azure/_data" .

set_build_images

# try upgrading just a single image to latest
FOCUS="\[ChangeImage\]\[Fake\]" TIMEOUT=50m ./hack/e2e.sh

# now upgrade the whole lot
ADMIN_MANIFEST=test/manifests/fakerp/admin-update.yaml make upgrade e2e
