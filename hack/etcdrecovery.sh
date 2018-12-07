#!/bin/bash

if [[ $# -lt 2 ]]; then
    echo error: usage "$0 resourceGroup backupblobname"
    exit 1
fi
export RESOURCEGROUP=$1
export RECOVER_ETCD_FROM_BACKUP=$2

if [[ $# -eq 0 && ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

echo running recovery with RECOVER_ETCD_FROM_BACKUP set to $RECOVER_ETCD_FROM_BACKUP
go generate ./...
go run cmd/recoveretcdcluster/recoveretcdcluster.go $RECOVER_ETCD_FROM_BACKUP
