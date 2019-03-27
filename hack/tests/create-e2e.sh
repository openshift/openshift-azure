#!/bin/bash -ex

prdetail="$(python -c 'import json, os; o=json.loads(os.environ["CLONEREFS_OPTIONS"]); print "%s-%s-" % (o["refs"][0]["pulls"][0]["author"].lower(), o["refs"][0]["pulls"][0]["number"])' 2>/dev/null || true)"
export RESOURCEGROUP="e2e-$prdetail$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)"

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

trap 'set +e; kill -15 ${MON_PID}; wait; make artifacts; make delete' EXIT

make create

make e2e
