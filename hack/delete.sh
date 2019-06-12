#!/bin/bash -e

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1
export OPENSHIFT_INSTALL_DATA=vendor/github.com/openshift/installer/data/data
go run cmd/azure/azure.go -n $RESOURCEGROUP -a "Delete"
