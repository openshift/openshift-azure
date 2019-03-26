#!/bin/bash -ex

export RESOURCEGROUP=e2e-$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)

cd $GOPATH/src/github.com/openshift/openshift-azure

## configure secrets
ln -s /usr/local/e2e-secrets/azure secrets
set +x
. secrets/secret
set -x

# start monitoring and record PID
make monitoring-build
./monitoring -outputdir=/tmp/artifacts &
MON_PID=$!

# prepare ci-operator env
. hack/tests/ci-operator-prepare.sh

trap 'kill -15 ${MON_PID}; wait; make artifacts; make delete' EXIT

make create

make e2e
