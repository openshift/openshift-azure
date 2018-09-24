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

NO_WAIT=${NO_WAIT:false}

if [[ "$NO_WAIT" == "true" ]]
then
	DELETE_FLAGS="$DELETE_FLAGS --no-wait"
fi

if [[ ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)

hack/dns.sh zone-delete $RESOURCEGROUP

rm -rf _data

az group delete -n $RESOURCEGROUP -y ${DELETE_FLAGS}
