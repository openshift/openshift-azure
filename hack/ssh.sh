#!/bin/bash -e

killagent() {
    [[ -n "$SSH_AGENT_PID" ]] && kill "$SSH_AGENT_PID"
}

if [[ $# -ne 2 ]]; then
    echo "usage: $0 resourcegroup {0,1,2}"
    exit 1
fi

RESOURCEGROUP="$1"
ID="$2"
shift 2

IP=$(az vmss list-instance-public-ips -g $RESOURCEGROUP -n ss-master --query "[$ID].ipAddress" | tr -d '"')

trap killagent EXIT

eval "$(ssh-agent)"
ssh-add _data/_out/id_rsa 2>/dev/null

ssh -A -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no \
    -o LogLevel=QUIET "$@" cloud-user@$IP
