#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo "usage: $0 source_version"
    exit 1
fi

cleanup() {
    set +e

    if [[ -n "$ARTIFACT_DIR" ]]; then
        exec &>"$ARTIFACT_DIR/cleanup"
    fi

    stop_monitoring
    make artifacts

    if [[ -n "$NO_DELETE" ]]; then
        return
    fi
    make delete
    git checkout "$GIT_CURRENT"  # restore git tree
    az group delete -g "$RESOURCEGROUP" --yes --no-wait
}
trap cleanup EXIT

. hack/tests/ci-operator-prepare.sh

# NOTE(ehashman): Without --abbrev-ref, restoring to the current commit results
# in detached HEAD state; this gets the current branch/tag
GIT_CURRENT="$(git rev-parse --abbrev-ref HEAD)"
GIT_TARGET="$1"

start_monitoring

git checkout $GIT_TARGET

make create
set_build_images

make e2e-scaleupdown
