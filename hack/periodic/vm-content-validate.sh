#!/bin/bash -e

cleanup() {
  set +e

  if [[ -n "$NO_DELETE" ]]; then
    return
  fi

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

[[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...

# TODO: Tag production images with separate tag and tests those
export IMAGE_RESOURCENAME=$(az image list -g $IMAGE_RESOURCEGROUP -o json --query "[?starts_with(name, '${DEPLOY_OS:-rhel7}-${DEPLOY_VERSION//v}') && tags.valid=='true'].name | sort(@) | [-1]" | tr -d '"')
export RESOURCEGROUP=$IMAGE_RESOURCENAME-validate
echo "Verify image $IMAGE_RESOURCENAME"
go run -ldflags "-X main.gitCommit=$GITCOMMIT" ./cmd/vmimage -imageResourceGroup "$IMAGE_RESOURCEGROUP" -image "$IMAGE_RESOURCENAME" -buildResourceGroup "$RESOURCEGROUP" -validate=true

UPDATE=false

if [[ $(cat /tmp/info |wc -l) -ge 3 ]]; then 
    UPDATE=true
fi

if [[ $(cat /tmp/check |wc -l) -ge 3 ]]; then 
    UPDATE=true
fi

if [ $UPDATE == true ]; then
# TODO: need to post file content into body of the payload
curl -X POST -u openshift-azure-robot:$GITHUB_TOKEN -H "Content-Type:application/json" --data @-  https://api.github.com/repos/openshift/openshift-azure/issues <<'EOF'
{
    "title": "VM Content update required",
    "body": "VM Image requires package update"
}
EOF
# TODO: Upload results to `images.repors` blob and add reference into the issue.
fi



