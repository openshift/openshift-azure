#!/bin/bash -ex

if [[ -z "$DEV_PULL_SECRET" ]]; then
    echo error: must set DEV_PULL_SECRET
    exit 1
fi

# Create docker dockerconfigjson auth string
AUTH=$(echo -n "notused:$DEV_PULL_SECRET" | base64 -w 0)
# Create k8s secret string
SECRET=$(echo "{\"auths\":{\"registry.reg-aws.openshift.com\":{\"auth\":\"${AUTH}\"}}}" | base64 -w 0 )
export KUBECONFIG=aks/admin.kubeconfig
# Create secret in HCP
sed "s/SECRET_PACEHOLDER/$SECRET/g" $(dirname $0)/secret.yaml | kubectl apply -f - -n $RESOURCEGROUP
# Patch SA to use secret
kubectl patch serviceaccount default -p '{"imagePullSecrets": [{"name": "devreg"}]}' -n  $RESOURCEGROUP 
