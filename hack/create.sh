#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

set -x

go generate ./...
if [[ -n "$TEST_IN_PRODUCTION" ]]; then
  go run cmd/createorupdate/createorupdate.go -use-prod=true
else
  go run cmd/createorupdate/createorupdate.go
fi

echo
echo  Cluster available at https://$RESOURCEGROUP.$AZURE_REGION.cloudapp.azure.com/
echo
