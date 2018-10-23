#!/bin/bash -x

if ! az account show >/dev/null; then
    exit 1
fi

if [[ -z "$DNS_DOMAIN" ]]; then
    echo error: must set DNS_DOMAIN
    exit 1
fi

if [[ -z "$DNS_RESOURCEGROUP" ]]; then
    echo error: must set DNS_RESOURCEGROUP
    exit 1
fi

if [[ "$NO_WAIT" == "true" ]]; then
	NO_WAIT_FLAG="--no-wait"
fi

if [[ $# -eq 0 && ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

if [[ -z "$AZURE_SUBSCRIPTION_ID" ]]; then
    AZURE_SUBSCRIPTION_ID=$(az account show --query id --output tsv)
fi

if [[ $# -eq 1 ]]; then
    export RESOURCEGROUP=$1
else
    export RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

USE_PROD_FLAG="-use-prod=true"
if [[ -z "$TEST_IN_PRODUCTION" ]]; then
    hack/dns.sh zone-delete $RESOURCEGROUP
    rm -rf _data
    USE_PROD_FLAG="-use-prod=false"
fi

go run cmd/createorupdate/createorupdate.go -request=DELETE $USE_PROD_FLAG
