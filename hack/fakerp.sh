#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

# kill any process holding 8080; should be useful in local development
fuser -k 8080/tcp || true

go generate ./...
go run cmd/fakerp/main.go
