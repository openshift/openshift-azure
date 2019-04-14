#!/bin/bash -e

cleanup() {
  set +e

  if [[ -n "$ARTIFACTS" ]]; then
    exec &>"$ARTIFACTS/cleanup"
  fi

  make artifacts

  if [[ -n "$NO_DELETE" ]]; then
    return
  fi
  make delete
  az group delete -g "$RESOURCEGROUP" --yes --no-wait
}
trap cleanup EXIT

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
go run -ldflags "-X main.gitCommit=$GITCOMMIT" ./cmd/vmimage -imageResourceGroup "$IMAGE_RESOURCEGROUP" -image "$IMAGE_RESOURCENAME" -imageStorageAccount "$IMAGE_STORAGEACCOUNT"

export AZURE_REGIONS=eastus
export RESOURCEGROUP="${IMAGE_RESOURCENAME//./}-e2e"

make create

make e2e

TAGS=$(az image show -g "$IMAGE_RESOURCEGROUP" -n "$IMAGE_RESOURCENAME" --query tags | python -c 'import sys; import json; tags = json.load(sys.stdin); print " ".join(["%s=%s" % (k, v) for (k, v) in tags.items()])')
az resource tag -g "$IMAGE_RESOURCEGROUP" -n $IMAGE_RESOURCENAME --resource-type Microsoft.Compute/images --tags $TAGS valid=true

echo "Built image $IMAGE_RESOURCENAME"
