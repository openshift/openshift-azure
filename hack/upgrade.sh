#!/bin/bash -ex

set +x
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

if [[ -z "$AZURE_SUBSCRIPTION_ID" ]]; then
    echo error: must set AZURE_SUBSCRIPTION_ID
    exit 1
fi

if [[ -z "$AZURE_TENANT_ID" ]]; then
    echo error: must set AZURE_TENANT_ID
    exit 1
fi

if [[ -z "$AZURE_CLIENT_ID" ]]; then
    echo error: must set AZURE_CLIENT_ID
    exit 1
fi

if [[ -z "$AZURE_CLIENT_SECRET" ]]; then
    echo error: must set AZURE_CLIENT_SECRET
    exit 1
fi
set -x

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

export RESOURCEGROUP=$1

go generate ./...
go run cmd/createorupdate/createorupdate.go -loglevel=debug
