#!/bin/bash -e

cleanup() {
  set +e

  generate_artifacts

  delete
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

if [ -z "$IMAGE_RESOURCENAME" ] ;
then
  IMAGE_RESOURCENAME=$(az image list -g images --query '[-1].name')
fi

IMAGE_RESOURCEGROUP="${IMAGE_RESOURCEGROUP:-images}"
IMAGE_STORAGEACCOUNT="${IMAGE_STORAGEACCOUNT:-openshiftimages}"

echo "Validating: ${IMAGE_RESOURCENAME}"

go generate ./... && go run -ldflags "-X main.gitCommit=$(git rev-parse --short=10 HEAD)" ./cmd/vmimage -imageResourceGroup "${IMAGE_RESOURCEGROUP}" -image "${IMAGE_RESOURCENAME}" -imageStorageAccount "${IMAGE_STORAGEACCOUNT}" -validate "true"

# Currently there are only 3 logs we want to capture
mv /tmp/{yum_update_info,yum_check_update,scap-report.html} ${ARTIFACTS:-/tmp}/
