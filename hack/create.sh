#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

set -x

USE_PROD_FLAG="-use-prod=false"
if [[ -n "$TEST_IN_PRODUCTION" ]]; then
    USE_PROD_FLAG="-use-prod=true"
else
    go generate ./...
    go run cmd/fakerp/main.go &
fi

trap 'set +ex; return_id=$?; kill $(lsof -t -i :8080); wait; exit $return_id' EXIT

go run cmd/createorupdate/createorupdate.go $USE_PROD_FLAG
