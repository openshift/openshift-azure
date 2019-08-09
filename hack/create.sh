#!/bin/bash -e

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

export RESOURCEGROUP=$1
# use image RG images to test latest code
export IMAGE_RESOURCEGROUP=images
export IMAGE_RESOURCENAME=$(az image list -g $IMAGE_RESOURCEGROUP -o json --query "[?starts_with(name, 'rhel7-${DEPLOY_VERSION//v}') && tags.valid=='true'].name | sort(@) | [-1]" | tr -d '"')


rm -rf _data
mkdir -p _data/_out

if [[ -n "$TEST_IN_PRODUCTION" ]]; then
    TEST_IN_PRODUCTION="-use-prod=true"
else
    [[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...
    go run cmd/fakerp/main.go &
fi
if [[ -n "$ADMIN_MANIFEST" ]]; then
    ADMIN_MANIFEST="-admin-manifest=$ADMIN_MANIFEST"
fi

trap 'return_id=$?; set +ex; kill $(lsof -t -i :8080); wait $(lsof -t -i :8080); exit $return_id' EXIT

go run cmd/createorupdate/createorupdate.go ${TEST_IN_PRODUCTION:-} ${ADMIN_MANIFEST:-}
