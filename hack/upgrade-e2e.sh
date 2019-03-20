#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
    echo error: $0 source_cluster_tag_version
    exit 1
fi

export RESOURCEGROUP=$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)
export SOURCE=$1

# start monitor from head and record pid
make monitoring-build
./monitoring -outputdir=/tmp/artifacts &
MON_PID=$!

T=$(mktemp -d)
export GOPATH=$T
trap "rm -rf $T" EXIT
git clone -b $SOURCE https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure

cd $T/src/github.com/openshift/openshift-azure

ln -s /usr/secrets secrets
set +x
. secrets/secret
set -x

trap 'kill -2 ${MON_PID}; wait; make delete' EXIT

make create

cd /go/src/github.com/openshift/openshift-azure

. hack/ci-operator-prepare.sh

cp -a $T/src/github.com/openshift/openshift-azure/_data .

make upgrade

make e2e
