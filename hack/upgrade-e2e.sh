#!/bin/bash -ex

if [[ $# -ne 2  ]]; then
    echo error: $0 resourcegroup source_cluster_tag_version
    exit 1
fi

export RESOURCEGROUP=$1
export SOURCE=tags/$2

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

    # iniciate source code for source cluster
    T="$(mktemp -d)"
    export GOPATH="${T}"
    trap "rm -rf ${T}" EXIT
    mkdir -p "${T}/src/github.com/openshift/"
    cd "${T}/src/github.com/openshift/"
    git clone https://github.com/openshift/openshift-azure
    cd openshift-azure
    git checkout $SOURCE
    # link shared secrets
    ln -s /usr/secrets secrets

    trap "make delete" EXIT
    echo "Create source cluster"
    make create

    # try upgrade cluster to newest code base
    export GOPATH="/go"
    # link secret
    ln -s /usr/secrets /go/src/github.com/openshift/openshift-azure/secrets
    ## copy manifest files
    ## TODO: fakeRP should read config blob so this should be removed
    cp -r ${T}/src/github.com/openshift/openshift-azure/_data /go/src/github.com/openshift/openshift-azure/

    cd /go/src/github.com/openshift/openshift-azure

    # set ci namespace images for cluster creation
    REGISTRY=registry.svc.ci.openshift.org
    export SYNC_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:sync
    export ETCDBACKUP_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:etcdbackup
    export AZURE_CONTROLLERS_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:azure-controllers
    export METRICSBRIDGE_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:metricsbridge
    export STARTUP_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:startup
    export TLSPROXY_IMAGE=${REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:tlsproxy
    echo "Upgrade cluster to master"
    ./hack/upgrade.sh $RESOURCEGROUP

    # test
    make e2e

else
    echo "Secrets not found, skip the run"
fi
