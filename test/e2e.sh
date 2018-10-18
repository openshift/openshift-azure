#!/bin/bash -ex

set +x
if ! az account show >/dev/null; then
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

if [[ -z "$AZURE_CLIENT_ID" ]]; then
    echo error: must set AZURE_CLIENT_ID
    exit 1
fi

if [[ -z "$AZURE_CLIENT_SECRET" ]]; then
    echo error: must set AZURE_CLIENT_SECRET
    exit 1
fi

if [[ -z "$AZURE_REGION" ]]; then
    echo error: must set AZURE_REGION
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
set -x

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1

rm -rf _data
mkdir -p _data/_out

set -x

# TEST_IN_PRODUCTION: (optional) whether to run using the prod RP or the fake RP
# MANIFEST: (optional) manifest to apply when creating a new cluster
# EXEC: (optional) command to execute once the cluster is created
# UPDATE: (optional) manifest to apply once the cluster is created and EXEC is done
# UPDATE_EXEC: (optional) command to execute once the cluster is updated
# ARTIFACT_DIR: (optional) directory to save cluster artifacts before the cluster gets cleaned up

go generate ./...

USE_PROD_FLAG="-use-prod=false"
if [[ -n "$TEST_IN_PRODUCTION" ]]; then
  USE_PROD_FLAG="-use-prod=true"
fi

EXEC_FLAG=""
if [[ -n "$EXEC" ]]; then
    EXEC_FLAG="-exec=$EXEC"
fi

UPDATE_FLAG=""
UPDATE_EXEC_FLAG=""
if [[ -n "$UPDATE_MANIFEST" ]]; then
    UPDATE_FLAG="-update=$UPDATE_MANIFEST"
    UPDATE_EXEC_FLAG="-update-exec=$UPDATE_EXEC"
fi

ARTIFACT_DIR_FLAG=""
ARTIFACT_KUBECONFIG_FLAG=""
if [[ -n "$ARTIFACT_DIR" ]]; then
    ARTIFACT_DIR_FLAG="-artifact-dir=$ARTIFACT_DIR"
    ARTIFACT_KUBECONFIG_FLAG="-artifact-kubeconfig=_data/_out/admin.kubeconfig"
fi

go run cmd/createorupdate/createorupdate.go -rm $USE_PROD_FLAG $EXEC_FLAG $UPDATE_FLAG $UPDATE_EXEC_FLAG $ARTIFACT_DIR_FLAG $ARTIFACT_KUBECONFIG_FLAG
