#!/bin/bash -e

if [[ -z "$DEV_PULL_SECRET" ]]; then
    echo error: must set DEV_PULL_SECRET
    exit 1
fi

if [[ -z "$DEV_REGISTRY" ]]; then
    echo error: must set DEV_REGISTRY
    exit 1
fi

if [[ -z "$KUBECONFIG" ]]; then
    echo error: must set KUBECONFIG
    exit 1
fi

if [[ -z "$RESOURCEGROUP" ]]; then
    echo error: must set RESOURCEGROUP
    exit 1
fi

kubectl create namespace $RESOURCEGROUP
# Create docker dockerconfigjson auth string
AUTH=$(echo -n "notused:$DEV_PULL_SECRET" | base64 -w 0)
# Create secret in HCP
kubectl apply -n $RESOURCEGROUP -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: pull-secret
stringData:
  .dockerconfigjson: '{"auths":{"$DEV_REGISTRY":{"auth":"$AUTH"}}}'
type: kubernetes.io/dockerconfigjson
EOF

# Patch SA to use secret
kubectl patch serviceaccount default -p '{"imagePullSecrets": [{"name": "pull-secret"}]}' -n $RESOURCEGROUP
