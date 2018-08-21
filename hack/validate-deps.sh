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

cp -a "${DIR}/.." "${T}/"

cd "${T}/"
glide up -v


if [[ -n "$(git status --porcelain)" ]]; then
	echo "glide update produced dependency changes that were not already present"
	echo "Run \"glide up -v\" to update the dependencies locally"
	exit 1
fi

echo "Dependencies have no material difference than what is committed."
