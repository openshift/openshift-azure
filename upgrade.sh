#!/bin/bash -ex

# To run this, you need:
# - to be logged in to the hosting cluster (oc login)
# - to have the AZURE_* environment variables set

RESOURCEGROUP=$(awk '/^ResourceGroup:/ { print $2 }' <_data/manifest.yaml)

go generate ./...
ImageResourceGroup=images ImageResourceName=centos7-3.10-201806231427 \
    go run cmd/createorupdate/createorupdate.go

# poor man's helm (without tiller running)
helm template pkg/helm/chart -f _data/_out/values.yaml --output-dir _data/_out
oc apply -Rf _data/_out/osa/templates

# TODO: need to apply ARM deployment changes

# will eventually run as an HCP pod, for development run it locally
KUBECONFIG=_data/admin.kubeconfig go run cmd/sync/sync.go

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/healthcheck/healthcheck.go
