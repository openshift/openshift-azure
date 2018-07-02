#!/bin/bash -ex

# To run this, you need:
# - aks/admin.kubeconfig for the hosting cluster
# - to be logged in to Azure (az login)
# - to have the AZURE_{SUBSCRIPTION,TENANT}_ID environment variables set

if [[ ! -e aks/admin.kubeconfig ]]; then
    echo error: aks/admin.kubeconfig must exist
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

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

az group create -n $RESOURCEGROUP -l eastus >/dev/null

if [[ -z "$AZURE_CLIENT_ID" ]]; then
    set +x
    . <(tools/aad.sh app-create openshift.$RESOURCEGROUP.osadev.cloud $RESOURCEGROUP)
    set -x
fi

# TODO: if the user interrupts the process here, the AAD application will leak.

cat >_data/manifest.yaml <<EOF
TenantID: $AZURE_TENANT_ID
SubscriptionID: $AZURE_SUBSCRIPTION_ID
ClientID: $AZURE_CLIENT_ID
ClientSecret: $AZURE_CLIENT_SECRET
Location: eastus
ResourceGroup: $RESOURCEGROUP
VMSize: Standard_D2s_v3
ComputeCount: 1
InfraCount: 1
PublicHostname: openshift.$RESOURCEGROUP.osadev.cloud
RoutingConfigSubdomain: $RESOURCEGROUP.osadev.cloud
EOF

go generate ./...
IMAGE=$(az image list -g images -o json --query "[?starts_with(name, 'centos7-3.10') && tags.valid=='true'].name | sort(@) | [-1]" | tr -d '"')
ImageResourceGroup=images ImageResourceName=$IMAGE \
    go run cmd/createorupdate/createorupdate.go

az group deployment create -g $RESOURCEGROUP -n azuredeploy --template-file _data/_out/azuredeploy.json --no-wait

# poor man's helm (without tiller running)
helm template pkg/helm/chart -f _data/_out/values.yaml --output-dir _data/_out
KUBECONFIG=aks/admin.kubeconfig kubectl create namespace $RESOURCEGROUP
KUBECONFIG=aks/admin.kubeconfig kubectl apply -n $RESOURCEGROUP -Rf _data/_out/osa/templates

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

# will eventually run as an HCP pod, for development run it locally
KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/sync/sync.go

az group deployment wait -g $RESOURCEGROUP -n azuredeploy --created --interval 10

ROUTERIP=$(az network public-ip list -g $RESOURCEGROUP --query "[?name == 'ip-router'].ipAddress | [0]" | tr -d '"')
tools/dns.sh a-create $RESOURCEGROUP '*' $ROUTERIP

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/healthcheck/healthcheck.go
