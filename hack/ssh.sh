#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 [ -n {0,1,2} ] [ -c command ] [ resourcegroup ]

EOF
    exit 1
}

cleanup() {
    [[ -n "$ID_RSA" ]] && rm -f "$ID_RSA"
    [[ -n "$SSH_AGENT_PID" ]] && kill "$SSH_AGENT_PID"
}

ID=0

while getopts :n:c: o; do
    case $o in
        n)
            ID=$OPTARG
            if [[ $ID != 0 && $ID != 1 && $ID != 2 ]]; then usage; fi
            ;;
        c)
            COMMAND=$OPTARG
            ;;
        *)
            usage
            ;;
    esac
done

if [[ -z "$AZURE_SUBSCRIPTION_ID" ]]; then
    AZURE_SUBSCRIPTION_ID=$(az account show --query id --output tsv)
fi

shift $((OPTIND-1))
RESOURCEGROUP=$1

trap cleanup EXIT

eval "$(ssh-agent)"

if [[ -z "$RESOURCEGROUP" ]]; then
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
    ssh-add _data/_out/id_rsa 2>/dev/null
	[[ -e _data/_out/id_rsa.old ]] && ssh-add _data/_out/id_rsa.old 2>/dev/null
fi

IP=$(az vmss list-instance-public-ips --subscription $AZURE_SUBSCRIPTION_ID -g $RESOURCEGROUP -n ss-master --query "[$ID].ipAddress" | tr -d '"')

ssh -A -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no cloud-user@$IP $COMMAND
