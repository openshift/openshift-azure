#!/bin/bash -ex

if [[ $# -ne 2 ]]; then
    echo error: $0 source_cluster_tag_version
    exit 1
fi

export RESOURCEGROUP=$(head -c6 </dev/urandom | base64)
export SOURCE=$1

T=$(mktemp -d)
export GOPATH=$T
trap "rm -rf $T" EXIT
git clone -b $SOURCE https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure

cd $T/src/github.com/openshift/openshift-azure

ln -s /usr/secrets secrets
set +x
. secrets/secret
set -x

trap 'make delete' EXIT
make create

cd /go/src/github.com/openshift/openshift-azure

. hack/ci-operator-prepare.sh

cp -a $T/src/github.com/openshift/openshift-azure/_data .

make upgrade

make e2e
