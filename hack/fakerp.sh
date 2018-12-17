#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

go generate ./...
go run cmd/fakerp/main.go
