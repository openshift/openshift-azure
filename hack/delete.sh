#!/bin/bash -x

if ! az account show >/dev/null; then
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

if [[ ! -e _data/manifest.yaml ]]; then
    echo error: _data/manifest.yaml must exist
    exit 1
fi

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1
PUBLICHOSTNAME=$(awk '/^  publicHostname:/ { print $2 }' <_data/manifest.yaml)

hack/dns.sh zone-delete $RESOURCEGROUP

rm -rf _data

az group delete -n $RESOURCEGROUP -y
