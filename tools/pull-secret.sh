#!/bin/bash -e

if [[ -z "$DEV_PULL_SECRET" ]]; then
    echo error: must set DEV_PULL_SECRET
    exit 1
fi

export KUBECONFIG=aks/admin.kubeconfig
kubectl create namespace $RESOURCEGROUP
# Create docker dockerconfigjson auth string
AUTH=$(echo -n "notused:$DEV_PULL_SECRET" | base64 -w 0)
# Create k8s secret string
SECRET=$(echo -n "{\"auths\":{\"${DEV_REGISTRY:-registry.reg-aws.openshift.com}\":{\"auth\":\"${AUTH}\"}}}" | base64 -w 0 )
# Create secret in HCP
kubectl apply -n $RESOURCEGROUP -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: pull-secret
data:
  .dockerconfigjson: "${SECRET}"
type: kubernetes.io/dockerconfigjson
EOF

# Patch SA to use secret
kubectl patch serviceaccount default -p '{"imagePullSecrets": [{"name": "pull-secret"}]}' -n  $RESOURCEGROUP 
