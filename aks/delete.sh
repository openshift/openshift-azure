#!/bin/bash -e

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1

az group delete -n $RESOURCEGROUP -y
