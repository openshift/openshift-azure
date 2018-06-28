#!/bin/bash -ex

# To run this, you need:
# - to be logged in to the hosting cluster (oc login)
# - the default service account in your namespace to be in the privileged SCC
#   (oc adm policy add-scc-to-user privileged system:serviceaccount:demo:default)
# - to be logged in to Azure (az login)
# - to have the AZURE_* environment variables set

RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)

go generate ./...
go run cmd/upgrade/upgrade.go

# poor man's helm (without tiller running)
helm template pkg/helm/chart -f _data/_out/values.yaml --output-dir _data/_out
oc apply -Rf _data/_out/osa/templates

# TODO: need to apply ARM deployment changes

# will eventually run as an HCP pod, for development run it locally
KUBECONFIG=_data/admin.kubeconfig go run cmd/sync/sync.go

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/health/health.go
