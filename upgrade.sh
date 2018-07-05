#!/bin/bash -ex

RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)

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
