#!/usr/bin/env bash

if [ "$1" == "" ]; then
    echo "plugin config file argument is missing"
	exit 1
fi
PLUGINCONFIG="$1"

####################################################
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
####################################################

# Validate if plugin config clusterVersion matches newest tag on the branch
# This runs only on release branches 
if [[ "$(git branch | grep \* | cut -d ' ' -f2)" =~ "release-" ]]; then
	echo "validating release branch "$(git branch | grep \* | cut -d ' ' -f2)
	CONFIG_VERSION=$(cat ${DIR}/../pluginconfig/${PLUGINCONFIG} | grep "clusterVersion:" | awk '{ print $2}')
	git tag -l | grep $CONFIG_VERSION
	if [ $? != 0 ]; then
		echo "plugin config release version ${CONFIG_VERSION} does not exist as tag"
		exit 1
	fi
else 
	echo "not a release branch. skip validation."
fi

