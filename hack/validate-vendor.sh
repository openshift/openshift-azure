#!/usr/bin/env bash

####################################################
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
####################################################

set -x

T="$(mktemp -d)"
trap "rm -rf ${T}" EXIT

mkdir -p $T/src/github.com/openshift/openshift-azure
cp -a "${DIR}/.." "${T}/src/github.com/openshift/openshift-azure"

GOPATH=$T go build github.com/openshift/openshift-azure/...
