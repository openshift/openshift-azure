#!/bin/bash -e

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

go run cmd/azure/azure.go -n $RESOURCEGROUP
