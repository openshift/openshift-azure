#!/bin/bash -ex

set_build_images() {
    if [[ ! -e /usr/local/e2e-secrets/azure ]]; then
        return
    fi

    export AZURE_IMAGE=registry.svc.ci.openshift.org/$OPENSHIFT_BUILD_NAMESPACE/stable:azure
}

start_monitoring() {
    make monitoring
    if [[ -n "$ARTIFACT_DIR" ]]; then
        outputdir="-outputdir=$ARTIFACT_DIR"
    fi

    if [ $# -eq 1 ]; then
        ./monitoring "$outputdir" -configfile=$1 &
    else
        ./monitoring "$outputdir" &
    fi
    MON_PID=$!
}

stop_monitoring() {
    if [[ -n "$MON_PID" ]]; then
        kill -15 "$MON_PID"
        wait
    fi
}

if [[ ! -e /usr/local/e2e-secrets/azure ]]; then
    return
fi

export ARTIFACT_DIR=/tmp/artifacts
export GOPATH=/go # our prow configuration overrides our image setting to /home/prow/go
export NO_WAIT=true
export RESOURCEGROUP_TTL=4h

mkdir -p $ARTIFACT_DIR

prdetail="$(python -c 'import json, os; o=json.loads(os.environ["CLONEREFS_OPTIONS"]); print "%s-%s-" % (o["refs"][0]["pulls"][0]["author"].lower(), o["refs"][0]["pulls"][0]["number"])' 2>/dev/null || true)"
export RESOURCEGROUP="$(basename "$0" .sh)-$prdetail$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)"

ln -sf /usr/local/e2e-secrets/azure secrets

set +x
. ./secrets/secret
export AZURE_CLIENT_ID="$AZURE_CI_CLIENT_ID"
export AZURE_CLIENT_SECRET="$AZURE_CI_CLIENT_SECRET"
export AZURE_LEGACY_MASTER_CLIENT_ID="$AZURE_CI_LEGACY_MASTER_CLIENT_ID"
export AZURE_LEGACY_MASTER_CLIENT_SECRET="$AZURE_CI_LEGACY_MASTER_CLIENT_SECRET"
export AZURE_LEGACY_WORKER_CLIENT_ID="$AZURE_CI_LEGACY_WORKER_CLIENT_ID"
export AZURE_LEGACY_WORKER_CLIENT_SECRET="$AZURE_CI_LEGACY_WORKER_CLIENT_SECRET"

az login --service-principal -u ${AZURE_CLIENT_ID} -p ${AZURE_CLIENT_SECRET} --tenant ${AZURE_TENANT_ID} >/dev/null
set -x
