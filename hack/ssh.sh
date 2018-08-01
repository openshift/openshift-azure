#!/bin/bash -e

if [[ $# -ne 3 ]]; then
    echo "usage: $0 resourcegroup ss-master|ss-infra|ss-compute {0,1,2,...}"
    exit 1
fi

RESOURCEGROUP=$1
SS=$2
ID=$3

IP=$(az vmss list-instance-public-ips -g $RESOURCEGROUP -n $SS --query "[$ID].ipAddress" | tr -d '"')

scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
    -o LogLevel=QUIET -i _data/_out/id_rsa _data/_out/id_rsa cloud-user@$IP:


exec ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
    -o LogLevel=QUIET -i _data/_out/id_rsa cloud-user@$IP
