#!/bin/bash -e

cleanup() {
  set +e

  ci_notify $PHASE
  generate_artifacts
  delete
}

trap cleanup EXIT

PHASE=image_setup
. hack/tests/ci-prepare.sh

TAG=$(git describe --tags HEAD)
if [[ $(git status --porcelain) = "" ]]; then
  GITCOMMIT="$TAG-clean"
else
  GITCOMMIT="$TAG-dirty"
fi

export IMAGE_RESOURCEGROUP="${IMAGE_RESOURCEGROUP:-images}"
export IMAGE_RESOURCENAME="${IMAGE_RESOURCENAME:-rhel7-3.11-$(TZ=Etc/UTC date +%Y%m%d%H%M)}"
export IMAGE_STORAGEACCOUNT="${IMAGE_STORAGEACCOUNT:-openshiftimages}"

[[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...
PHASE=image_build
go run -ldflags "-X main.gitCommit=$GITCOMMIT" ./cmd/vmimage -imageResourceGroup "$IMAGE_RESOURCEGROUP" -image "$IMAGE_RESOURCENAME" -imageStorageAccount "$IMAGE_STORAGEACCOUNT"

export AZURE_REGIONS=eastus
if [[ -z "$RESOURCEGROUP" ]]; then
  export RESOURCEGROUP="${IMAGE_RESOURCENAME//./}-e2e"
fi

PHASE=image_cluster_build
make create

PHASE=image_e2e
make e2e

PHASE=image_validation
make vmimage-validate

PHASE=image_tag
TAGS=$(az image show -g "$IMAGE_RESOURCEGROUP" -n "$IMAGE_RESOURCENAME" --query tags | python -c 'import sys; import json; tags = json.load(sys.stdin); print " ".join(["%s=%s" % (k, v) for (k, v) in tags.items()])')
az resource tag -g "$IMAGE_RESOURCEGROUP" -n $IMAGE_RESOURCENAME --resource-type Microsoft.Compute/images --tags $TAGS valid=true

echo "Built image ${IMAGE_RESOURCENAME}"
PHASE=build_complete
