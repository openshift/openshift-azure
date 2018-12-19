#!/bin/bash -x

if [[ $# -eq 0 && ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

if [[ $# -eq 1 ]]; then
    export RESOURCEGROUP=$1
else
    export RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

USE_PROD_FLAG="-use-prod=false"
if [[ -n "$TEST_IN_PRODUCTION" ]]; then
    USE_PROD_FLAG="-use-prod=true"
else
    go generate ./...
    go run cmd/fakerp/main.go &
fi

trap 'set +ex; return_id=$?; kill $(lsof -t -i :8080); wait; exit $return_id' EXIT

go run cmd/createorupdate/createorupdate.go -request=DELETE $USE_PROD_FLAG
