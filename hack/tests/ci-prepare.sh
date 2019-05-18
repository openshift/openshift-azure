#!/bin/bash -e

set_build_images() {
    if [[ ! -e /var/run/secrets/kubernetes.io ]]; then
        return
    fi

    export AZURE_IMAGE=quay.io/openshift-on-azure/ci-azure:$(git describe --tags HEAD)
    make azure-image
}

start_monitoring() {
    make monitoring
    if [[ -n "$ARTIFACTS" ]]; then
        outputdir="-outputdir=$ARTIFACTS"
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

generate_artifacts() {
  if [[ -n "$ARTIFACTS" ]]; then
      exec &>"$ARTIFACTS/cleanup"
  fi

  stop_monitoring

  make artifacts
}

delete() {
  realrp="$1"

  if [[ -n "$NO_DELETE" ]]; then
      return
  fi

  # only delete for fakerp
  if [[ -z "$realrp" ]]; then
    make delete
  fi

  az group delete -g "$RESOURCEGROUP" --yes --no-wait
}

if [[ ! -e /var/run/secrets/kubernetes.io ]]; then
    return
fi

export NO_WAIT=true
export RESOURCEGROUP_TTL=4h

mkdir -p $ARTIFACTS

pullnumber="$(python -c 'import json, os; o=json.loads(os.environ["JOB_SPEC"]); print "%s-" % o["refs"]["pulls"][0]["number"]' 2>/dev/null || true)"
export RESOURCEGROUP="ci-$pullnumber$(basename "$0" .sh)-$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)"

echo "RESOURCEGROUP is $RESOURCEGROUP"
echo

make secrets

. ./secrets/secret
export AZURE_CLIENT_ID="$AZURE_CI_CLIENT_ID"
export AZURE_CLIENT_SECRET="$AZURE_CI_CLIENT_SECRET"
export AZURE_LEGACY_MASTER_CLIENT_ID="$AZURE_CI_LEGACY_MASTER_CLIENT_ID"
export AZURE_LEGACY_MASTER_CLIENT_SECRET="$AZURE_CI_LEGACY_MASTER_CLIENT_SECRET"
export AZURE_LEGACY_WORKER_CLIENT_ID="$AZURE_CI_LEGACY_WORKER_CLIENT_ID"
export AZURE_LEGACY_WORKER_CLIENT_SECRET="$AZURE_CI_LEGACY_WORKER_CLIENT_SECRET"

az login --service-principal -u ${AZURE_CLIENT_ID} -p ${AZURE_CLIENT_SECRET} --tenant ${AZURE_TENANT_ID} >/dev/null
