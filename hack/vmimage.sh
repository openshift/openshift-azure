#!/bin/bash -e

cleanup() {
  set +e

  generate_artifacts

  # delete the image validation build_resource_group
  az group delete -g "${BUILD_RESOURCE_GROUP}" --yes --no-wait

  delete
}

trap cleanup EXIT

. hack/tests/ci-prepare.sh

TAG=$(git describe --tags HEAD)
if [[ $(git status --porcelain) = "" ]]; then
  GITCOMMIT="$TAG-clean"
else
  GITCOMMIT="$TAG-dirty"
fi

export BUILD_RESOURCE_GROUP="vmimage-$(date +%Y%m%d%H%M)"
export IMAGE_RESOURCEGROUP="${IMAGE_RESOURCEGROUP:-images}"
export IMAGE_RESOURCENAME="${IMAGE_RESOURCENAME:-rhel7-3.11-$(TZ=Etc/UTC date +%Y%m%d%H%M)}"
export IMAGE_STORAGEACCOUNT="${IMAGE_STORAGEACCOUNT:-openshiftimages}"

[[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...
go run -ldflags "-X main.gitCommit=$GITCOMMIT" ./cmd/vmimage -imageResourceGroup "$IMAGE_RESOURCEGROUP" -image "$IMAGE_RESOURCENAME" -imageStorageAccount "$IMAGE_STORAGEACCOUNT"

export AZURE_REGIONS=eastus
if [[ -z "$RESOURCEGROUP" ]]; then
  export RESOURCEGROUP="${IMAGE_RESOURCENAME//./}-e2e"
fi

make create

make e2e

# Run the image validation and capture the artifacts
go generate ./... && go run -ldflags "-X main.gitCommit=$GITCOMMIT" ./cmd/vmimage -buildResourceGroup "${BUILD_RESOURCE_GROUP}" -imageResourceGroup "${IMAGE_RESOURCEGROUP}" -image "${IMAGE_RESOURCENAME}" -imageStorageAccount "${IMAGE_STORAGEACCOUNT}" -validate -preserveBuildResourceGroup

# Currently there are only 3 logs we want to capture
mv /tmp/{yum_update_info,yum_check_update,scap-report.html} ${ARTIFACTS:-/tmp}/ || true

az account show

TAGS=$(az image show -g "$IMAGE_RESOURCEGROUP" -n "$IMAGE_RESOURCENAME" --query tags | python -c 'import sys; import json; tags = json.load(sys.stdin); print " ".join(["%s=%s" % (k, v) for (k, v) in tags.items()])')
az resource tag -g "$IMAGE_RESOURCEGROUP" -n $IMAGE_RESOURCENAME --resource-type Microsoft.Compute/images --tags $TAGS valid=true

echo "Built image $IMAGE_RESOURCENAME"
