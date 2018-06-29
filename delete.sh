#!/bin/bash -x

# To run this, you need:
# - to be logged in to the hosting cluster (oc login)
# - to be logged in to Azure (az login)
# - to have the AZURE_* environment variables set

PUBLICHOSTNAME=$(awk '/^PublicHostname:/ { print $2 }' <_data/manifest.yaml)
RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)

# poor man's helm (without tiller running)
oc delete -Rf _data/_out/osa/templates

rm -rf _data

tools/dns.sh zone-delete $RESOURCEGROUP

tools/aad.sh app-delete $PUBLICHOSTNAME

az group delete -n $RESOURCEGROUP -y
