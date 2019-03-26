#!/bin/bash -ex

COMMIT=$(git rev-parse --short HEAD)
if [[ $(git status --porcelain) = "" ]]; then
  COMMIT="$COMMIT-clean"
else
  COMMIT="$COMMIT-dirty"
fi

export IMAGE_RESOURCEGROUP="${IMAGE_RESOURCEGROUP:-images}"
export IMAGE_RESOURCENAME="${IMAGE_RESOURCENAME:-rhel7-3.11-$(TZ=Etc/UTC date +%Y%m%d%H%M)}"

go generate ./...
go run -ldflags "-X main.gitCommit=$COMMIT" ./cmd/vmimage -imageResourceGroup "$IMAGE_RESOURCEGROUP" -image "$IMAGE_RESOURCENAME"

export AZURE_REGION=eastus
export RESOURCEGROUP="${IMAGE_RESOURCENAME//./}-e2e"

trap '[[ -z "$NO_DELETE" ]] && make artifacts; make delete' EXIT

make create

make e2e

TAGS=$(az image show -g "$IMAGE_RESOURCEGROUP" -n "$IMAGE_RESOURCENAME" --query tags | python -c 'import sys; import json; tags = json.load(sys.stdin); print " ".join(["%s=%s" % (k, v) for (k, v) in tags.items()])')
az resource tag -g "$IMAGE_RESOURCEGROUP" -n $IMAGE_RESOURCENAME --resource-type Microsoft.Compute/images --tags $TAGS valid=true

echo "Built image $IMAGE_RESOURCENAME"
