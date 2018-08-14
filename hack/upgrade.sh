#!/bin/bash -ex

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

if [[ ! -e _data/_out/admin.kubeconfig ]]; then
    echo error: _data/_out/admin.kubeconfig must exist
    exit 1
fi

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

go generate ./...
go run cmd/createorupdate/createorupdate.go

export KUBECONFIG=_data/_out/admin.kubeconfig

# TODO: not all ARM template changes are going to be applied with vmssrollout
# upgrade master scale set
go run cmd/vmssrollout/vmssrollout.go -subscription $AZURE_SUBSCRIPTION_ID \
                                      -resource-group $RESOURCEGROUP \
                                      -name ss-master \
                                      -config _data/containerservice.yaml \
                                      -old-config _data/containerservice_old.yaml \
                                      -template-file _data/_out/azuredeploy.json \
                                      -old-template-file _data/_out/azuredeploy_old.json \
                                      -in-place \
                                      -role master

# upgrade infra scale set
go run cmd/vmssrollout/vmssrollout.go -subscription $AZURE_SUBSCRIPTION_ID \
                                      -resource-group $RESOURCEGROUP \
                                      -name ss-infra \
                                      -config _data/containerservice.yaml \
                                      -old-config _data/containerservice_old.yaml \
                                      -template-file _data/_out/azuredeploy.json \
                                      -old-template-file _data/_out/azuredeploy_old.json \
                                      -drain \
                                      -role infra

# upgrade compute scale set
go run cmd/vmssrollout/vmssrollout.go -subscription $AZURE_SUBSCRIPTION_ID \
                                      -resource-group $RESOURCEGROUP \
                                      -name ss-compute \
                                      -config _data/containerservice.yaml \
                                      -old-config _data/containerservice_old.yaml \
                                      -template-file _data/_out/azuredeploy.json \
                                      -old-template-file _data/_out/azuredeploy_old.json \
                                      -drain \
                                      -role compute

if [[ "$RUN_SYNC_LOCAL" == "true" ]]; then
    # will eventually run as an HCP pod, for development run it locally
    KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/sync/sync.go -run-once=true
fi

KUBECONFIG=_data/_out/admin.kubeconfig go run cmd/healthcheck/healthcheck.go
