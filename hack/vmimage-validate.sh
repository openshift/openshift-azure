#!/bin/bash -e

cleanup() {
  set +e

  az group delete -g "${BUILD_RESOURCE_GROUP}" --yes --no-wait
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

BUILD_RESOURCE_GROUP="vmimage-$(date +%Y%m%d%H%M)"
IMAGE_RESOURCEGROUP="${IMAGE_RESOURCEGROUP:-images}"
IMAGE_STORAGEACCOUNT="${IMAGE_STORAGEACCOUNT:-openshiftimages}"

if [ -z "$IMAGE_RESOURCENAME" ] ;
then
  IMAGE_RESOURCENAME=$(az image list -g $IMAGE_RESOURCEGROUP --query '[-1].name' -o tsv)
fi

echo "Validating: ${IMAGE_RESOURCENAME}"

# pass -preserveBuildResourceGroup so we can copy artifacts
go generate ./... && go run -ldflags "-X main.gitCommit=$(git rev-parse --short=10 HEAD)" ./cmd/vmimage -buildResourceGroup "${BUILD_RESOURCE_GROUP}" -imageResourceGroup "${IMAGE_RESOURCEGROUP}" -image "${IMAGE_RESOURCENAME}" -imageStorageAccount "${IMAGE_STORAGEACCOUNT}" -validate -preserveBuildResourceGroup

# Logs we want to capture
mv /tmp/{yum_updateinfo,yum_updateinfo_list_security,yum_check_update,scap_report.html} ${ARTIFACTS:-/tmp}/ || true

if grep -q "rule-result rule-result-fail" ${ARTIFACTS:-/tmp}/scap_report.html; then
  >&2 echo "Some SCAP rules failed. Check report in the build artifacts"
  exit 1
fi

if [ -s "${ARTIFACTS:-/tmp}/yum_updateinfo_list_security" ]; then
  >&2 echo "Found pending security updates:"
  >&2 cat /tmp/yum_updateinfo_list_security
  exit 1
fi
