#!/bin/bash -e

if [[ $# -ne 1 ]]; then
    echo "usage: $0 vm-infra-0|vm-compute-0"
    exit 1
fi

RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)
HOST=$1

IP=$(az vm list-ip-addresses -g $RESOURCEGROUP -n $HOST --query '[0].virtualMachine.network.publicIpAddresses[0].ipAddress' | tr -d '"')

ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=QUIET \
    -i _data/_out/id_rsa cloud-user@$IP
