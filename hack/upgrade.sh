#!/bin/bash -ex

if ! az account show >/dev/null; then
    exit 1
fi

if [[ $# -eq 0 && ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

if [[ $# -eq 1 ]]; then
    export RESOURCEGROUP=$1
else
    export RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

USE_PROD_FLAG="-use-prod=false"
if [[ -n "$TEST_IN_PRODUCTION" ]]; then
    USE_PROD_FLAG="-use-prod=true"
fi

if [[ -n "$MANIFEST" ]]; then
    cat $MANIFEST | envsubst > _data/manifest.yaml
fi

sed -i '/provisioningState/d' _data/manifest.yaml
go generate ./...
go run cmd/createorupdate/createorupdate.go -timeout 1h $USE_PROD_FLAG
