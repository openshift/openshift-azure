#!/bin/bash -x

if ! az account show >/dev/null; then
    exit 1
fi

ENVIRONMENT_CONFIG=$(dirname $(dirname $0))/env

# check if the environment config file exists
if [[ ! -f ${ENVIRONMENT_CONFIG} ]]; then
	echo error: must setup an env config file in project root
	exit 1
fi

# source the environment config file
. ${ENVIRONMENT_CONFIG}

if [[ -z "$DNS_DOMAIN" ]]; then
    echo error: must set DNS_DOMAIN
    exit 1
fi

if [[ -z "$DNS_RESOURCEGROUP" ]]; then
    echo error: must set DNS_RESOURCEGROUP
    exit 1
fi

if [[ ! -e _data/containerservice.yaml ]]; then
    echo error: _data/containerservice.yaml must exist
    exit 1
fi

RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)

hack/dns.sh zone-delete $RESOURCEGROUP

rm -rf _data

az group delete -n $RESOURCEGROUP -y
