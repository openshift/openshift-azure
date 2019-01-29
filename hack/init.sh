#!/bin/bash -ex

if [[ $# -ne 1 ]]; then
else
    export SOURCE=tags/$1
fi

# remove all existing code. Script will be running from "cache"
# this implies all other script and release steps are backwards compatible
# or cherry-picked to old branches if changes are made
set -x
cd $GOPATH/src/github.com/openshift
git clone https://github.com/mjudeikis/openshift-azure openshift-azure-source
cd openshift-azure-source
git checkout "$SOURCE"
