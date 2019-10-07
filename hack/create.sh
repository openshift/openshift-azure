#!/bin/bash -e

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

if [[ -n "$API_VERSION" ]]; then
    API_VERSION="-api-version=$API_VERSION"
fi

[[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...
# building takes time, so if we use "go run" we will not get the PID of
# the server as it won't have started yet.
go build ./cmd/fakerp
./fakerp &
trap 'return_id=$?; set +ex; kill $(lsof -t -i :8080); wait $(lsof -t -i :8080); exit $return_id' EXIT

if [[ -n "$ADMIN_MANIFEST" ]]; then
    ADMIN_MANIFEST="-admin-manifest=$ADMIN_MANIFEST"
fi

go run cmd/createorupdate/createorupdate.go ${API_VERSION:-} ${ADMIN_MANIFEST:-}
