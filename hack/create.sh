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

if [[ -z "$AZURE_REGION" ]]; then
    echo error: must set AZURE_REGION
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
set -x

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

# if AZURE_CLIENT_ID is used as AZURE_AAD_CLIENT_ID, script will reset global team account!
set +x
if [[ "$AZURE_AAD_CLIENT_ID" && "$AZURE_AAD_CLIENT_ID" != "$AZURE_CLIENT_ID" ]]; then
    . <(hack/aad.sh app-update $AZURE_AAD_CLIENT_ID https://$RESOURCEGROUP.$AZURE_REGION.cloudapp.azure.com/oauth2callback/Azure%20AD)
else
    AZURE_AAD_CLIENT_ID=$AZURE_CLIENT_ID
    AZURE_AAD_CLIENT_SECRET=$AZURE_CLIENT_SECRET
fi
export AZURE_AAD_CLIENT_ID
export AZURE_AAD_CLIENT_SECRET
set -x

if [[ -z "$MANIFEST" ]]; then
    MANIFEST="test/manifests/normal/create.yaml"
fi
cat $MANIFEST | envsubst > _data/manifest.yaml

go generate ./...
if [[ -n "$TEST_IN_PRODUCTION" ]]; then
  go run cmd/createorupdate/createorupdate.go -use-prod=true
else
  go run cmd/createorupdate/createorupdate.go
fi

echo
echo  Cluster available at https://$RESOURCEGROUP.$AZURE_REGION.cloudapp.azure.com/
echo
