#!/bin/bash -ex

if [[ $# -eq 0 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1

rm -rf _in _out
mkdir _in _out

cat >_in/manifest <<EOF
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
cp _in/manifest _out/manifest

go generate ./...
go run cmd/create/create.go

helm template pkg/helm/chart -f _out/values.yaml --output-dir _out

# poor man's helm (without tiller running)
oc delete -Rf _out/osa/templates || true
oc create -Rf _out/osa/templates

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
az group deployment create -g $RESOURCEGROUP --template-file _out/azuredeploy.json

# will eventually run as an HCP pod, for development run it locally
KUBECONFIG=_out/admin.kubeconfig go run cmd/sync/sync.go

# TODO: health check
