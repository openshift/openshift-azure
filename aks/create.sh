#!/bin/bash -e

if [[ -z "$AZURE_CLIENT_ID" ]]; then
    echo error: must set AZURE_CLIENT_ID and AZURE_CLIENT_SECRET environment variables
    exit 1
fi

if [[ $# -ne 2 ]]; then
    echo usage: $0 resourcegroup '~/.ssh/id_rsa.pub'
    exit 1
fi

RESOURCEGROUP=$1
SSHKEYPATH=$2

az group create -n $RESOURCEGROUP -l eastus >/dev/null
az group deployment create -g $RESOURCEGROUP \
    --template-file $(dirname $0)/azuredeploy.json \
    --parameters clientId=$AZURE_CLIENT_ID \
    --parameters clientSecret=$AZURE_CLIENT_SECRET \
    --parameters keyData="$(cat $SSHKEYPATH)" \
    >/dev/null

az aks get-credentials -g $RESOURCEGROUP -n aks -f - >$(dirname $0)/admin.kubeconfig

KUBECONFIG=$(dirname $0)/admin.kubeconfig kubectl create -f $(dirname $0)/ingress-nginx.yaml
KUBECONFIG=$(dirname $0)/admin.kubeconfig kubectl create -f $(dirname $0)/tiller.yaml
