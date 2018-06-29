#!/bin/bash -ex

# To run this, you need:
# - to be logged in to the hosting cluster (oc login)
# - to be logged in to Azure (az login)
# - to have the AZURE_* environment variables set

if [ -z "$AZURE_CLIENT_ID" ]; then
    echo error: must set AZURE_* environment variables
    exit 1
fi

if [[ $# -eq 0 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

cat >_data/manifest.yaml <<EOF
TenantID: $AZURE_TENANT_ID
SubscriptionID: $AZURE_SUBSCRIPTION_ID
ClientID: $AZURE_CLIENT_ID
ClientSecret: $AZURE_CLIENT_SECRET
Location: eastus
ResourceGroup: $RESOURCEGROUP
VMSize: Standard_D4s_v3
ComputeCount: 1
InfraCount: 1
ImageResourceGroup: images
ImageResourceName: centos7-3.10-201806231427
PublicHostname: openshift.$RESOURCEGROUP.osadev.cloud
RoutingConfigSubdomain: $RESOURCEGROUP.osadev.cloud
EOF

NAMESPACE=$(oc project -q)
oc adm policy add-scc-to-user privileged system:serviceaccount:$NAMESPACE:default

go generate ./...
go run cmd/create/create.go

# poor man's helm (without tiller running)
helm template pkg/helm/chart -f _data/_out/values.yaml --output-dir _data/_out
oc create -Rf _data/_out/osa/templates

while true; do
    MASTERIP=$(oc get service master-api -o template --template '{{ if .status.loadBalancer }}{{ (index .status.loadBalancer.ingress 0).ip }}{{ end }}')
    if [[ -n "$MASTERIP" ]]; then
        break
    fi
    sleep 1
done

tools/dns.sh zone-create $RESOURCEGROUP
tools/dns.sh a-create $RESOURCEGROUP openshift $MASTERIP
# when we know the router IP, do tools/dns.sh a-create $RESOURCEGROUP '*' $ROUTERIP

az group create -n $RESOURCEGROUP -l eastus
az group deployment create -g $RESOURCEGROUP --template-file _data/_out/azuredeploy.json

# will eventually run as an HCP pod, for development run it locally
KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/sync/sync.go

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/health/health.go
