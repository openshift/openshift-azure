#!/bin/bash -ex

if [[ $# -ne 1  ]]; then
    echo error: $0 resourcegroup 
    exit 1
fi

export RESOURCEGROUP=$1

if [[ -f /usr/local/e2e-secrets/azure/secret ]] ;then
    set +x
    source /usr/local/e2e-secrets/azure/secret
    export AZURE_AAD_CLIENT_ID=$AZURE_CLIENT_ID
    export AZURE_AAD_CLIENT_SECRET=$AZURE_CLIENT_SECRET
    set -x
    export DNS_DOMAIN=osadev.cloud
    export DNS_RESOURCEGROUP=dns
    export DEPLOY_VERSION=v3.11
    export NO_WAIT=true

    export BASE_CODE_DIR=/home/prow/go/src/github.com/openshift

    # configure both copied of the code with secrets
    ln -s /usr/local/e2e-secrets/azure ${BASE_CODE_DIR}/openshift-azure-old/secrets
    ln -s /usr/local/e2e-secrets/azure ${BASE_CODE_DIR}/openshift-azure-new/secrets

    # enable old code
    ln -s ${BASE_CODE_DIR}/openshift-azure-old ${BASE_CODE_DIR}/openshift-azure

    # create cluster using old code
    trap "./hack/delete.sh $RESOURCEGROUP" EXIT
    echo "Create source cluster"
    cd ${BASE_CODE_DIR}/openshift-azure
    ./hack/create.sh $RESOURCEGROUP

    # enable new code
    ln -sf ${BASE_CODE_DIR}openshift-azure-new ${BASE_CODE_DIR}/openshift-azure

    # copy manifest files
    # TODO: fakeRP should read config blob so this should be removed
    cp -r ${BASE_CODE_DIR}/openshift-azure-old/_data ${BASE_CODE_DIR}/openshift-azure-new/
    cd ${BASE_CODE_DIR}/openshift-azure
    ./hack/upgrade.sh $RESOURCEGROUP

else
    echo "This scipt can only be ran inside CI pod"
fi
