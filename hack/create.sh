#!/bin/bash -ex

cleanup(){
	trap '' TERM
	kill 0
	wait
}
trap cleanup EXIT

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

set -x

if [[ -n "$TEST_IN_PRODUCTION" ]]; then
    TEST_IN_PRODUCTION="-use-prod=true"
else
    go generate ./...
    go run cmd/fakerp/main.go &
fi
if [[ -n "$ADMIN_MANIFEST" ]]; then
    ADMIN_MANIFEST="-admin-manifest=$ADMIN_MANIFEST"
fi

go run cmd/createorupdate/createorupdate.go ${TEST_IN_PRODUCTION:-} ${ADMIN_MANIFEST:-}
