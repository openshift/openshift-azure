#!/bin/bash -ex

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

source_secret(){
    # running in ci
    if [[ -f /usr/local/e2e-secrets/azure/secret ]];then
        set +x
        . /usr/local/e2e-secrets/azure/secret
        set -x
    else
        # running locally
        set +x
        . secrets/secret
        set -x
    fi
}

link_secret(){
    # running in ci
    if [[ -f /usr/local/e2e-secrets/azure/secret ]];then
        ln -s /usr/local/e2e-secrets/azure secrets
    else
        # running locally
        ln -s ${ORG_GOPATH}/src/github.com/openshift/openshift-azure/secrets secrets
    fi
}

preprare_ci_env(){
    # running in ci
    if [[ -f /usr/local/e2e-secrets/azure/secret ]];then
       . hack/tests/ci-operator-prepare.sh
    fi
}

if [[ $# -ne 1 ]]; then
    echo error: $0 source_cluster_tag_version
    exit 1
fi

export RESOURCEGROUP=upgrade-$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)
export SOURCE=$1

source_secret

export ORG_GOPATH=$GOPATH
T=$(mktemp -d)

# start monitor from head and record pid
make monitoring-build
./monitoring -outputdir=/tmp/artifacts -configfile=$T/src/github.com/openshift/openshift-azure/_data/containerservice.yaml &
MON_PID=$!

export GOPATH=$T
git clone -b $SOURCE https://github.com/openshift/openshift-azure.git $T/src/github.com/openshift/openshift-azure

cd $T/src/github.com/openshift/openshift-azure

link_secret

trap 'set +e; kill -15 ${MON_PID}; wait; make artifacts; make delete; rm -rf $T' EXIT

make create

export GOPATH=${ORG_GOPATH}
cd ${GOPATH}/src/github.com/openshift/openshift-azure

preprare_ci_env
link_secret

cp -a $T/src/github.com/openshift/openshift-azure/_data .

make upgrade

make e2e
