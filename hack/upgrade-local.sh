#!/bin/bash -ex
# Upgrade-local script works like upgrade-e2e CI script
# It is intended to run on local development environment only

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

ORG_GOPATH=$GOPATH

T=$(mktemp -d)
export GOPATH=$T
trap "rm -rf $T" EXIT
git clone -b $SOURCE https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure

cd $T/src/github.com/openshift/openshift-azure

ln -s ${ORG_GOPATH}/src/github.com/openshift/openshift-azure/secrets secrets
set +x
. secrets/secret
set -x

trap 'kill -2 ${MON_PID}; wait; make delete' EXIT

make create

export GOPATH=${ORG_GOPATH}
cd ${GOPATH}/src/github.com/openshift/openshift-azure

cp -a $T/src/github.com/openshift/openshift-azure/_data .

make upgrade

make e2e
