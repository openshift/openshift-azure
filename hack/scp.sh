#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 [ -n {0,1,2} ] [ -s file_sourcen] [ -d file_destination] [ resourcegroup ]

EOF
    exit 1
}

cleanup() {
    [[ -n "$ID_RSA" ]] && rm -f "$ID_RSA"
    [[ -n "$SSH_AGENT_PID" ]] && kill "$SSH_AGENT_PID"
}

ID=0

while getopts :n:s:d: o; do
    case $o in
        n)
            ID=$OPTARG
            if [[ $ID != 0 && $ID != 1 && $ID != 2 ]]; then usage; fi
            ;;
        s)
            FILE_SRC=$OPTARG
            ;;
        d)
            FILE_DST=$OPTARG
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

ID_RSA=$(mktemp)
chmod 0600 $ID_RSA

if [[ -z "$RESOURCEGROUP" ]]; then
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
    cat _data/_out/id_rsa >$ID_RSA
fi

IP=$(az vmss list-instance-public-ips --subscription $AZURE_SUBSCRIPTION_ID -g $RESOURCEGROUP -n ss-master --query "[$ID].ipAddress" | tr -d '"')

eval "$(ssh-agent)"
ssh-add $ID_RSA 2>/dev/null

scp -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no cloud-user@$IP:$FILE_SRC $FILE_DST
