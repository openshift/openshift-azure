#!/bin/bash -ex
# Upgrade-local script works like upgrade-e2e CI script
# It is intended to run on local development environment only

### Script flow:
# Script should follow upgrade-e2e.sh flow. 
# This script is for development/test only and might need tweaking to work
# on local environment.

if [[ $# -ne 1 ]]; then
    echo error: $0 source_cluster_tag_version
    exit 1
fi

export RESOURCEGROUP=$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)
export SOURCE=$1

set +x
. secrets/secret
set -x

ORG_GOPATH=$GOPATH
T=$(mktemp -d)
trap "rm -rf $T" EXIT INT

# start monitor from head and record pid
make monitoring-build
./monitoring -outputdir=/tmp/artifacts -configfile=$T/src/github.com/openshift/openshift-azure/_data/containerservice.yaml &
MON_PID=$!

export GOPATH=$T
git clone -b $SOURCE https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure

cd $T/src/github.com/openshift/openshift-azure

ln -s ${ORG_GOPATH}/src/github.com/openshift/openshift-azure/secrets secrets

trap 'kill -15 ${MON_PID}; wait; make delete' EXIT INT

make create

export GOPATH=${ORG_GOPATH}
cd ${GOPATH}/src/github.com/openshift/openshift-azure

make upgrade

make e2e
