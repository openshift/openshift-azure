#!/bin/bash -e

cleanup() {
    [[ -n "$ID_RSA" ]] && rm -f "$ID_RSA"
    [[ -n "$SSH_AGENT_PID" ]] && kill "$SSH_AGENT_PID"
}

trap cleanup EXIT

eval "$(ssh-agent | grep -v '^echo ')"

if [[ ! -z "$RESOURCEGROUP" ]]; then
    ssh-add _data/clusters/${RESOURCEGROUP}/id_rsa 2>/dev/null
else
	echo "${RESOURCEGROUP} not set"
fi

FULL_RESOURCEGROUP=$(cat _data/clusters/${RESOURCEGROUP}/metadata.json | jq -r .infraID)

portid=0
if [[ ! "$1" -eq "" ]]; then
    portid=$1
fi

IP=$(az network public-ip list -g ${FULL_RESOURCEGROUP}-rg | jq -r '.[] | select(.name == "'$FULL_RESOURCEGROUP'-pip") | .ipAddress')

ssh -v -A -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o LogLevel=ERROR -p 220${portid} -l core $IP
