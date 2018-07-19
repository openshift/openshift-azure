#!/bin/bash -ex

if [[ ! -e aks/admin.kubeconfig ]]; then
    echo error: aks/admin.kubeconfig must exist
    exit 1
fi

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
    . <(tools/aad.sh app-create openshift.$RESOURCEGROUP.$DNS_DOMAIN $RESOURCEGROUP)
    set -x
fi

# TODO: if the user interrupts the process here, the AAD application will leak.

cat >_data/manifest.yaml <<EOF
name: openshift
location: eastus
properties:
  openShiftVersion: "$DEPLOY_VERSION"
  publicHostname: openshift.$RESOURCEGROUP.$DNS_DOMAIN
  routingConfigSubdomain: $RESOURCEGROUP.$DNS_DOMAIN
  agentPoolProfiles:
  - name: compute
    count: 1
    vmSize: Standard_D2s_v3
  - name: infra
    count: 1
    vmSize: Standard_D2s_v3
    role: infra
  servicePrincipalProfile:
    clientID: $AZURE_CLIENT_ID
    secret: $AZURE_CLIENT_SECRET
EOF

go generate ./...
go run cmd/createorupdate/createorupdate.go

az group deployment create -g $RESOURCEGROUP -n azuredeploy --template-file _data/_out/azuredeploy.json --no-wait

if [[ ! -z "$DEV_PULL_SECRET" ]]; then
    KUBECONFIG=aks/admin.kubeconfig tools/pull-secret.sh
fi

KUBECONFIG=aks/admin.kubeconfig helm install --namespace $RESOURCEGROUP pkg/helm/chart -f _data/_out/values.yaml -n $RESOURCEGROUP >/dev/null

while true; do
    HCPINGRESSIP=$(KUBECONFIG=aks/admin.kubeconfig kubectl get ingress -n $RESOURCEGROUP master-api -o template --template '{{ if .status.loadBalancer }}{{ (index .status.loadBalancer.ingress 0).ip }}{{ end }}')
    if [[ -n "$HCPINGRESSIP" ]]; then
        break
    fi
    sleep 1
done

tools/dns.sh zone-create $RESOURCEGROUP
tools/dns.sh a-create $RESOURCEGROUP openshift $HCPINGRESSIP
tools/dns.sh a-create $RESOURCEGROUP openshift-tunnel $HCPINGRESSIP

if [[ "$RUN_SYNC_LOCAL" == "true" ]]; then
    # will eventually run as an HCP pod, for development run it locally
    KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/sync/sync.go -run-once=true
fi

az group deployment wait -g $RESOURCEGROUP -n azuredeploy --created --interval 10

ROUTERIP=$(az network public-ip list -g $RESOURCEGROUP --query "[?name == 'ip-router'].ipAddress | [0]" | tr -d '"')
tools/dns.sh a-create $RESOURCEGROUP '*' $ROUTERIP

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/healthcheck/healthcheck.go
