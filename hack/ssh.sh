#!/bin/bash -e

# ssh.sh is intended to behave very similarly to ssh: specify the master or
# worker hostname that you want to connect to, along with any other ssh options
# you want

cleanup() {
    [[ -n "$ID_RSA" ]] && rm -f "$ID_RSA"
    [[ -n "$SSH_AGENT_PID" ]] && kill "$SSH_AGENT_PID"
}

trap cleanup EXIT

eval "$(ssh-agent | grep -v '^echo ')"

if [[ -z "$RESOURCEGROUP" ]]; then
    # RESOURCEGROUP not specified, read from _data/containerservice.yaml
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
    ssh-add _data/_out/id_rsa 2>/dev/null
    # id_rsa.old exists during/after key rotation
	[[ -e _data/_out/id_rsa.old ]] && ssh-add _data/_out/id_rsa.old 2>/dev/null
else
    # get key from the cluster's config blob
	ID_RSA=$(mktemp)
	hack/config.sh get-config $RESOURCEGROUP | jq -r .config.sshKey | base64 -d >$ID_RSA
	ssh-add $ID_RSA 2>/dev/null
fi

opts=()
didsub=0
while [[ $1 ]]; do
    # look for the first string that feasibly matches a hostname and substitute it
    if [[ "$didsub" -eq 0 && "$1" =~ ^master-00000[012]$ ]]; then
        # masters: connect direct
        id=${1: -1}
        opts+=("$(az vmss list-instance-public-ips -g $RESOURCEGROUP -n ss-master --query "[$id].ipAddress" | tr -d '"')")
        didsub=1
    elif [[ "$didsub" -eq 0 && "$1" =~ ^[a-z0-9]{1,12}-[0-9]+-[a-z0-9]{6}$ ]]; then
        # workers: proxy via master-000000
        opts=("-o" "ProxyCommand=ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR -l cloud-user -W %h:%p $(az vmss list-instance-public-ips -g $RESOURCEGROUP -n ss-master --query "[0].ipAddress" | tr -d '"')" "$1" "${opts[@]}")
    else
        opts+=("$1")
    fi
    shift
done

if [[ ${#opts[@]} -eq 0 ]]; then
    # sop to previous behaviour
    opts=("$(az vmss list-instance-public-ips -g $RESOURCEGROUP -n ss-master --query "[0].ipAddress" | tr -d '"')")
fi

ssh -A -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR -l cloud-user "${opts[@]}"
