#!/bin/bash -ex
# upgrade-e2e script is intended to run in CI environmet and will not run localy

### Script flow:
# 1. Source credentials from secret
# 2. Create temporary folder for "source cluster" code
# 3. Start monitoring shim layer from master
#    We pass config file from temporary folder, 
#    because it will be created there first during source cluster creation
# 4. Link secrets to appropriate location
# 5. Clone released code and create a cluster
# 6. Switch to PR code base and inicate upgrade.
#    Old cluster upgarde is always managed by new or same version of RP

if [[ $# -ne 1 ]]; then
    echo error: $0 source_cluster_tag_version
    exit 1
fi

export RESOURCEGROUP=$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)
export SOURCE=$1

# source secrets early in the process
set +x
. /usr/local/e2e-secrets/azure/secret
set -x

T=$(mktemp -d)
trap "rm -rf $T" EXIT INT

# start monitor from head and record pid
make monitoring-build
./monitoring -outputdir=/tmp/artifacts -configfile=$T/src/github.com/openshift/openshift-azure/_data/containerservice.yaml &
MON_PID=$!

export GOPATH=$T
git clone -b $SOURCE https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure

cd $T/src/github.com/openshift/openshift-azure

ln -s /usr/local/e2e-secrets/azure secrets

trap 'kill -15 ${MON_PID}; wait; make delete' EXIT INT

make create

cd /go/src/github.com/openshift/openshift-azure

. hack/ci-operator-prepare.sh

make upgrade

make e2e
