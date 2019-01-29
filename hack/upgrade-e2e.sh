#!/bin/bash -ex

if [[ $# < 2 ]]; then
    echo error: $0 resourcegroup source [target]
    exit 1
fi

if [[ -f /usr/local/e2e-secrets/azure/secret ]] ;then
    set +x
    source /usr/local/e2e-secrets/azure/secret
    export AZURE_AAD_CLIENT_ID=$AZURE_CLIENT_ID
    export AZURE_AAD_CLIENT_SECRET=$AZURE_CLIENT_SECRET
    set -x

    export RESOURCEGROUP=$1
    export SOURCE=tags/$2
    if [[ -n "$3" ]]; then
        export TARGET=tag/$3
    fi

    # check-out target code base for cluster creation
    S="$(mktemp -d)"
    GOPATH="${S}"
    trap "rm -rf ${S}" EXIT
    mkdir -p "${S}/src/github.com/openshift/"
    cd "${S}/src/github.com/openshift/"
    git clone https://github.com/mjudeikis/openshift-azure
    cd openshift-azure
    git checkout "$SOURCE"

    # if we run in CI default location for ci-secret exist
    ln -s /usr/local/e2e-secrets/azure $PWD/secrets
    export DNS_DOMAIN=osadev.cloud
    export DNS_RESOURCEGROUP=dns
    export DEPLOY_VERSION=v3.11
    export NO_WAIT=true

    trap "./hack/delete.sh $RESOURCEGROUP" EXIT
    echo "Create source cluster"
    ./hack/create.sh $RESOURCEGROUP

    # init upgrade from master branch
    GOPATH="/home/prow/go/"
    cd ${GOPATH}/src/github.com/openshift/openshift-azure
    ln -s /usr/local/e2e-secrets/azure $PWD/secrets

    # copy manifest files
    # TODO: fakeRP should read config blob so this should be removed
    cp -r ${S}/src/github.com/openshift/openshift-azure/_data ${GOPATH}/src/github.com/openshift/openshift-azure/
    ls -la
    ./hack/upgrade.sh $RESOURCEGROUP

else
    echo "This scipt can only be ran inside CI pod"
fi
