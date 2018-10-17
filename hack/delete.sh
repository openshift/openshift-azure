#!/bin/bash -x

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

go run cmd/createorupdate/createorupdate.go -request=DELETE $USE_PROD_FLAG
