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

if [[ "$NO_WAIT" == "true" ]]; then
	NO_WAIT_FLAG="--no-wait"
fi

if [[ ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

if [[ $# -eq 1 ]]; then
    export RESOURCEGROUP=$1
else
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

hack/dns.sh zone-delete $RESOURCEGROUP

rm -rf _data

az group delete -n $RESOURCEGROUP -y ${NO_WAIT_FLAG}
