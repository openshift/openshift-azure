#!/bin/bash -ex

cleanup() {
    kill $(jobs -p) &>/dev/null || true
    wait
}

trap cleanup EXIT

if [[ $# -ne 2 ]]; then
    echo error: $0 resourcegroup blobname
    exit 1
fi

export RESOURCEGROUP=$1
export BLOBNAME=$2

go generate ./...
go run cmd/fakerp/main.go &

go run cmd/createorupdate/createorupdate.go -timeout 1h -restore-from-blob=$BLOBNAME
