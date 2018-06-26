#!/bin/bash -x

if [[ $# -eq 0 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1

# poor man's helm (without tiller running)
oc delete -Rf _out/osa/templates

rm -rf _in _out

tools/dns.sh zone-delete $RESOURCEGROUP

az group delete -n $RESOURCEGROUP -y
