#!/bin/bash -ex

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

if [[ -z "$DNS_DOMAIN" ]]; then
    echo error: must set DNS_DOMAIN
    exit 1
fi

if [[ -z "$DNS_RESOURCEGROUP" ]]; then
    echo error: must set DNS_RESOURCEGROUP
    exit 1
fi

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

az group create -n $RESOURCEGROUP -l eastus >/dev/null

if [[ -z "$AZURE_CLIENT_ID" ]]; then
    set +x
    . <(hack/aad.sh app-create openshift.$RESOURCEGROUP.$DNS_DOMAIN $RESOURCEGROUP)
    set -x
fi

# TODO: if the user interrupts the process here, the AAD application will leak.

cat >_data/manifest.yaml <<EOF
name: openshift
location: eastus
properties:
  openShiftVersion: "$DEPLOY_VERSION"
  publicHostname: openshift.$RESOURCEGROUP.$DNS_DOMAIN
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
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: compute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  servicePrincipalProfile:
    clientID: $AZURE_CLIENT_ID
    secret: $AZURE_CLIENT_SECRET
EOF

go generate ./...
go run cmd/createorupdate/createorupdate.go -loglevel=debug

az group deployment create -g $RESOURCEGROUP -n azuredeploy --template-file _data/_out/azuredeploy.json --no-wait

if [[ "$RUN_SYNC_LOCAL" == "true" ]]; then
    # will eventually run as an HCP pod, for development run it locally
    # sleep until FQDN/healthz responds with OK. It takes up to 5 min for API server to respond. 
    FQDN=$(awk '/^  fqdn:/ { print $2 }' <_data/manifest.yaml)
    while [[ "$(curl -s -k https://${FQDN}/healthz/ready)" != "ok" ]]; do sleep 5; done
    KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/sync/sync.go -run-once=true -loglevel=debug
fi

az group deployment wait -g $RESOURCEGROUP -n azuredeploy --created --interval 10

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/healthcheck/healthcheck.go -loglevel=debug

# TODO: This should be configured by MS
hack/dns.sh zone-create $RESOURCEGROUP
hack/dns.sh cname-create $RESOURCEGROUP openshift $RESOURCEGROUP.eastus.cloudapp.azure.com
hack/dns.sh cname-create $RESOURCEGROUP '*' $RESOURCEGROUP-router.eastus.cloudapp.azure.com

