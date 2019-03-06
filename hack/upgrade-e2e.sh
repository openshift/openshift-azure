#!/bin/bash -ex

if [[ $# == 1  ]]; then
    echo "pr code base to latest upgrade test"
fi

if [[ $# == 2  ]]; then
    echo "source code base to pr code base upgrade test"
    export SOURCE=tags/$2
fi

export RESOURCEGROUP=$1

if [[ -f /usr/secrets/secret ]] ;then
    set +x
    source  /usr/secrets/secret
    export AZURE_AAD_CLIENT_ID=$AZURE_CLIENT_ID
    export AZURE_AAD_CLIENT_SECRET=$AZURE_CLIENT_SECRET
    set -x
    export DNS_DOMAIN=osadev.cloud
    export DNS_RESOURCEGROUP=dns
    export DEPLOY_VERSION=v3.11
    export NO_WAIT=true

    export GOPATH="/go"
    cd ${GOPATH}/src/github.com/openshift/openshift-azure

    # if this is source code base to pr code upgrade
    if [[ $SOURCE != "" ]]; then
        git checkout $SOURCE
    fi
    # link shared secrets
    ln -s /usr/secrets /go/src/github.com/openshift/openshift-azure/secrets
    trap "make delete" EXIT
    echo "Create source cluster"
    make create

    # set ci namespace images for cluster creation
    REGISTRY=registry.svc.ci.openshift.org
    export SYNC_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:sync
    export ETCDBACKUP_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:etcdbackup
    export AZURE_CONTROLLERS_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:azure-controllers
    export METRICSBRIDGE_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:metricsbridge
    export STARTUP_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:startup
    echo "Upgrade cluster to master"
    ./hack/upgrade.sh $RESOURCEGROUP

    # test
    make e2e

else
    echo "Secrets not found, skip the run"
fi
