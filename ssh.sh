#!/bin/bash -e

if [[ $# -ne 2 ]]; then
    echo "usage: $0 ss-infra|ss-compute {0,1,2,...}"
    exit 1
fi

RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)
HOST=$1
ID=$2

IP=$(az vmss list-instance-public-ips -g ${RESOURCEGROUP} -n $1 --query '['$2'].ipAddress' | tr -d '"')

ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=QUIET \
    -i _data/_out/id_rsa cloud-user@$IP
