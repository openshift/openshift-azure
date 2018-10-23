#!/bin/bash -ex

set +x
if ! az account show >/dev/null; then
    exit 1
fi

if [[ -z "$AZURE_SUBSCRIPTION_ID" ]]; then
    echo error: must set AZURE_SUBSCRIPTION_ID
    exit 1
fi

if [[ -z "$AZURE_TENANT_ID" ]]; then
    echo error: must set AZURE_TENANT_ID
    exit 1
fi

if [[ -z "$AZURE_CLIENT_ID" ]]; then
    echo error: must set AZURE_CLIENT_ID
    exit 1
fi

if [[ -z "$AZURE_CLIENT_SECRET" ]]; then
    echo error: must set AZURE_CLIENT_SECRET
    exit 1
fi
set -x

if [[ -z "$DNS_DOMAIN" ]]; then
    echo error: must set DNS_DOMAIN
    exit 1
fi

if [[ -z "$DNS_RESOURCEGROUP" ]]; then
    echo error: must set DNS_RESOURCEGROUP
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
sed -i '/provisioningState/d' _data/manifest.yaml

go generate ./...
go run cmd/createorupdate/createorupdate.go
