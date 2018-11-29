#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

if [[ -n "$TEST_IN_PRODUCTION" ]]; then
  USE_PROD="-use-prod=true"
fi

cp test/manifests/normal/create.yaml _data/manifest.yaml
go generate ./...
go run cmd/createorupdate/createorupdate.go "${USE_PROD:-}"

echo
echo  Cluster available at https://$RESOURCEGROUP.$AZURE_REGION.cloudapp.azure.com/
echo
