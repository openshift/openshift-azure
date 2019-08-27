#!/bin/bash

if [[ $# -eq 0 && ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

if [[ $# -eq 1 ]]; then
    export RESOURCEGROUP=$1
else
    export RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

if [[ -n "$TEST_IN_PRODUCTION" ]]; then
    TEST_IN_PRODUCTION="-use-prod=true"
else
    [[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...
    # building takes time, so if we use "go run" we will not get the PID of
    # the server as it won't have started yet.
    go build ./cmd/fakerp
    ./fakerp &
    trap 'return_id=$?; set +ex; kill $(lsof -t -i :8080); wait $(lsof -t -i :8080); exit $return_id' EXIT
fi

go run cmd/createorupdate/createorupdate.go -request=DELETE ${TEST_IN_PRODUCTION:-}
RESULT=$?
if [[ $RESULT -eq 0 ]]; then
    rm -rf _data
fi
exit $RESULT
