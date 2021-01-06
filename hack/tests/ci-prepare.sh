#!/bin/bash -e

# turn off xtrace if enabled
#xtrace=false
#if echo $SHELLOPTS | egrep -q ':?xtrace:?'; then
#  xtrace=true
#  set +x
#fi

reset_xtrace() {
    if $xtrace; then
        set -x
    else
        set +x
    fi
}

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
  if [[ -n "$NO_DELETE" ]]; then
      return
  fi
  make delete
}

ci_notify() {
  # We create notifications in CI and only for periodic jobs. This will open an issue
  # with a link to a failed CI build.
  if [[ -e /var/run/secrets/kubernetes.io ]] && [ "${JOB_TYPE}" == "periodic" ]; then
    if [ $PHASE == "build_complete" ]; then
      ARGS="-success"
    fi
    go run ./hack/ci-notify/main.go -job-name "${JOB_NAME}" -comment "Phase: $1" $ARGS
  fi
}

if [[ ! -e /var/run/secrets/kubernetes.io ]]; then
    reset_xtrace
else
    mkdir -p  $(pwd)/secrets
    cp -R /secrets $(pwd)/
    chown -R $HOST_USER_ID:$HOST_GROUP_ID $(pwd)/secrets
fi

export NO_WAIT=true
export RESOURCEGROUP_TTL=4h

mkdir -p $ARTIFACTS

pullnumber="$(python -c 'import json, os; o=json.loads(os.environ["JOB_SPEC"]); print "%s-" % o["refs"]["pulls"][0]["number"]' 2>/dev/null || true)"
export RESOURCEGROUP="ci-$pullnumber$(basename "$0" .sh)-$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)"

echo "RESOURCEGROUP is $RESOURCEGROUP"
echo


. ./secrets/secret

reset_xtrace
