#!/bin/bash -ex

if [[ ! -e aks/admin.kubeconfig ]]; then
    echo error: aks/admin.kubeconfig must exist
    exit 1
fi

if [[ -z "$AZURE_SUBSCRIPTION_ID" ]]; then
    echo error: must set AZURE_SUBSCRIPTION_ID
    exit 1
fi

if [[ -z "$AZURE_TENANT_ID" ]]; then
    echo error: must set AZURE_TENANT_ID
    exit 1
fi

if [[ -z "$DNS_DOMAIN" ]]; then
    echo error: must set DNS_DOMAIN
    exit 1
fi

if [[ -z "$DNS_RESOURCEGROUP" ]]; then
    echo error: must set DNS_RESOURCEGROUP
    exit 1
fi

if [[ ! -e _data/manifest.yaml ]]; then
    echo error: _data/manifest.yaml must exist
    exit 1
fi

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

go generate ./...
go run cmd/createorupdate/createorupdate.go

KUBECONFIG=aks/admin.kubeconfig helm upgrade $RESOURCEGROUP pkg/helm/chart -f _data/_out/values.yaml >/dev/null

# TODO: verify the kubectl rollout status code below, not convinced that it's
# working properly

# TODO: when sync runs as an HCP pod (i.e. not in development), hopefully should
# be able to use helm upgrade --wait here
for d in master-etcd master-api master-controllers; do
    KUBECONFIG=aks/admin.kubeconfig kubectl rollout status deployment $d -n $RESOURCEGROUP -w
done

# TODO: need to apply ARM deployment changes

# will eventually run as an HCP pod, for development run it locally
KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/sync/sync.go

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/healthcheck/healthcheck.go
