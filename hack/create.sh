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

az group create -n $RESOURCEGROUP -l eastus --tags now=$(date +%s) >/dev/null

set +x
if [[ "$AZURE_AAD_CLIENT_ID" ]]; then
    . <(hack/aad.sh app-update $AZURE_AAD_CLIENT_ID https://openshift.${RESOURCEGROUP}.${DNS_DOMAIN}/oauth2callback/Azure%20AD)
else
    AZURE_AAD_CLIENT_ID=$AZURE_CLIENT_ID
    AZURE_AAD_CLIENT_SECRET=$AZURE_CLIENT_SECRET
fi
set -x

cat >_data/manifest.yaml <<EOF
name: openshift
location: eastus
properties:
  openShiftVersion: "$DEPLOY_VERSION"
  publicHostname: openshift.$RESOURCEGROUP.$DNS_DOMAIN
  fqdn: $RESOURCEGROUP.eastus.cloudapp.azure.com
  authProfile:
    identityProviders:
    - name: Azure AAD
      provider:
        kind: AADIdentityProvider
        clientId: $AZURE_AAD_CLIENT_ID
        secret: $AZURE_AAD_CLIENT_SECRET
  routerProfiles:
  - name: default
    publicSubdomain: $RESOURCEGROUP.$DNS_DOMAIN
  masterPoolProfile:
    name: master
    count: 3
    vmSize: Standard_D2s_v3
    osType: Linux
  agentPoolProfiles:
  - name: infra
    role: infra
    count: 2
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: compute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  servicePrincipalProfile:
    clientId: $AZURE_CLIENT_ID
    secret: $AZURE_CLIENT_SECRET
EOF

go generate ./...
go run cmd/createorupdate/*.go -loglevel=debug

# TODO: This should be configured by MS
hack/dns.sh zone-create $RESOURCEGROUP
hack/dns.sh cname-create $RESOURCEGROUP openshift $RESOURCEGROUP.eastus.cloudapp.azure.com
hack/dns.sh cname-create $RESOURCEGROUP '*' $RESOURCEGROUP-router.eastus.cloudapp.azure.com

