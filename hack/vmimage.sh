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

. hack/tests/ci-operator-prepare.sh

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
if [[ -z "$VERIFY" ]]; then
  go run -ldflags "-X main.gitCommit=$GITCOMMIT" ./cmd/vmimage -imageResourceGroup "$IMAGE_RESOURCEGROUP" -image "$IMAGE_RESOURCENAME" -imageStorageAccount "$IMAGE_STORAGEACCOUNT"

  export AZURE_REGIONS=eastus
  if [[ -z "$RESOURCEGROUP" ]]; then
    export RESOURCEGROUP="${IMAGE_RESOURCENAME//./}-e2e"
  fi

  make create

  make e2e

  TAGS=$(az image show -g "$IMAGE_RESOURCEGROUP" -n "$IMAGE_RESOURCENAME" --query tags | python -c 'import sys; import json; tags = json.load(sys.stdin); print " ".join(["%s=%s" % (k, v) for (k, v) in tags.items()])')
  az resource tag -g "$IMAGE_RESOURCEGROUP" -n $IMAGE_RESOURCENAME --resource-type Microsoft.Compute/images --tags $TAGS valid=true

  echo "Built image $IMAGE_RESOURCENAME"
else
  # get newest image
  # TODO: need to check only published images. Tags?
  export IMAGE_RESOURCENAME=$(az image list -g $IMAGE_RESOURCEGROUP -o json --query "[?starts_with(name, '${DEPLOY_OS:-rhel7}-${DEPLOY_VERSION//v}') && tags.valid=='true'].name | sort(@) | [-1]" | tr -d '"')
  export RESOURCEGROUP=$IMAGE_RESOURCENAME-validate
  echo "Verify image $IMAGE_RESOURCENAME"
  go run -ldflags "-X main.gitCommit=$GITCOMMIT" ./cmd/vmimage -imageResourceGroup "$IMAGE_RESOURCEGROUP" -image "$IMAGE_REOURCENAME" -validate=true

  cat /tmp/info
  cat /tmp/check
fi
