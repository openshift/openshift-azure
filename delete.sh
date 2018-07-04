#!/bin/bash -x

if [[ ! -e _data/manifest.yaml ]]; then
    echo error: _data/manifest.yaml must exist
    exit 1
fi

PUBLICHOSTNAME=$(awk '/^PublicHostname:/ { print $2 }' <_data/manifest.yaml)
RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)

KUBECONFIG=aks/admin.kubeconfig kubectl delete namespace $RESOURCEGROUP

tools/dns.sh zone-delete $RESOURCEGROUP

tools/aad.sh app-delete $PUBLICHOSTNAME

rm -rf _data

az group delete -n $RESOURCEGROUP -y
