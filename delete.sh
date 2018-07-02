#!/bin/bash -x

# To run this, you need:
# - aks/admin.kubeconfig for the hosting cluster
# - to be logged in to Azure (az login)

PUBLICHOSTNAME=$(awk '/^PublicHostname:/ { print $2 }' <_data/manifest.yaml)
RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)

KUBECONFIG=aks/admin.kubeconfig kubectl delete namespace $RESOURCEGROUP

tools/dns.sh zone-delete $RESOURCEGROUP

tools/aad.sh app-delete $PUBLICHOSTNAME

rm -rf _data

az group delete -n $RESOURCEGROUP -y
