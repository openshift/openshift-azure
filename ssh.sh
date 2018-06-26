#!/bin/bash -e

if [[ $# -eq 0 ]]; then
    echo "usage: $0 resourcegroup vm-infra-0|vm-compute-0"
    exit 1
fi

RESOURCEGROUP=$1
HOST=$2

IP=$(az vm list-ip-addresses -g $RESOURCEGROUP -n $HOST --query '[0].virtualMachine.network.publicIpAddresses[0].ipAddress' | tr -d '"')

ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=QUIET \
    -i _out/id_rsa cloud-user@$IP
