#!/bin/bash

set -eo pipefail

echo "SUITE=>${SUITE}"
if [[ -z "$RESOURCEGROUP" ]]; then
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

if [[ -n "$ARTIFACT_DIR" ]]; then
  ARTIFACT_FLAG="-artifact-dir=$ARTIFACT_DIR"
fi


# TODO: figure out env variables and other information required by cucumber
/usr/bin/scl enable rh-git29 rh-ror50 /opt/rh/rh-ruby24/root/usr/local/bin/cucumber
