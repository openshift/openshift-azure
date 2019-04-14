#!/bin/bash -e

if [[ $# -ne 2 ]]; then
  cat <<EOF >&2
usage:

$0 path/to/Dockerfile [repo/]image:tag
EOF
  exit 1
fi

DOCKERFILE=$1                      # e.g. images/azure/Dockerfile
IMAGE=$2                           # e.g. quay.io/openshift-on-azure/azure:v4.0-141-gf5682846

if [[ ! -e /var/run/secrets/kubernetes.io ]]; then
  docker pull $(awk '/^FROM/ {print $2}' <$DOCKERFILE)

  go get github.com/openshift/imagebuilder/cmd/imagebuilder
  ${GOPATH:-$HOME/go}/bin/imagebuilder -f $DOCKERFILE -t $IMAGE .

else
  [[ $IMAGE =~ ([^/]*)/([^:]*):(.*) ]]

  REPONAME=${BASH_REMATCH[1]}      # e.g. quay.io
  IMAGEPATH=${BASH_REMATCH[2]}     # e.g. openshift-on-azure/azure
  IMAGENAME=$(basename $IMAGEPATH) # e.g. azure
  TAG=${BASH_REMATCH[3]}           # e.g. v4.0-141-gf5682846

  if [[ $(curl -so /dev/null -w '%{http_code}' https://$REPONAME/v2/$IMAGEPATH/manifests/$TAG) == 200 ]]; then
    exit 0
  fi

  oc create -f - &>/dev/null <<EOF || rv=$?
apiVersion: build.openshift.io/v1
kind: BuildConfig
metadata:
  name: $IMAGENAME-$TAG
  namespace: azure
spec:
  output:
    to:
      kind: DockerImage
      name: $IMAGE
    pushSecret:
      name: openshift-on-azure-scratch-secret
  source:
    type: Binary
  strategy:
    dockerStrategy:
      dockerfilePath: $DOCKERFILE
      forcePull: true
      imageOptimizationPolicy: SkipLayers
    type: Docker
EOF

  if [[ $rv -eq 0 ]]; then
    trap "oc delete -n azure buildconfig $IMAGENAME-$TAG" EXIT
    # TODO: specifying '$DOCKERFILE azure' here is a hack, but is much faster than specifying '.'
    tar -cz $DOCKERFILE azure | oc start-build -n azure $IMAGENAME-$TAG --from-archive=- -F

    if [[ $(curl -so /dev/null -w '%{http_code}' https://$REPONAME/v2/$IMAGEPATH/manifests/$TAG) != 200 ]]; then
      exit 1
    fi

  else
    for ((i=0; i<60; i++)); do
      if [[ $(curl -so /dev/null -w '%{http_code}' https://$REPONAME/v2/$IMAGEPATH/manifests/$TAG) == 200 ]]; then
        exit 0
      fi
      sleep 10
    done
    exit 1
  fi
fi
