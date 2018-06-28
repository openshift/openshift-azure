#!/bin/bash -x

# To run this, you need:
# - to be logged in to the hosting cluster (oc login)
# - to be logged in to Azure (az login)
# - to have the AZURE_* environment variables set

if [[ $# -eq 0 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1

# poor man's helm (without tiller running)
oc delete -Rf _data/_out/osa/templates

rm -rf _data

tools/dns.sh zone-delete $RESOURCEGROUP

az group delete -n $RESOURCEGROUP -y
