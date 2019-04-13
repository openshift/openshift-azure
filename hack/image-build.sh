#!/bin/bash -ex

if [[ $# -ne 2 ]]; then
  cat <<EOF >&2
usage:

$0 path/to/Dockerfile [repo/]image:tag
EOF
  exit 1
fi

DOCKERFILE=$1                  # e.g. images/azure/Dockerfile

IMAGE=$2                       # e.g. [quay.io/openshift-on-azure/]azure:v4.0-141-gf5682846
IMAGENAME=$(basename ${2%%:*}) # e.g. azure
TAG=${2//*:}                   # e.g. v4.0-141-gf5682846

if [[ ! -e /usr/local/e2e-secrets/azure ]]; then
  docker pull $(awk '/^FROM/ {print $2}' <$DOCKERFILE)

  go get github.com/openshift/imagebuilder/cmd/imagebuilder
  ${GOPATH:-$HOME/go}/bin/imagebuilder -f $DOCKERFILE -t $IMAGE .

else
  oc create -f - &>/dev/null <<EOF || true
apiVersion: image.openshift.io/v1
kind: ImageStream
metadata:
  name: $IMAGENAME
EOF

  oc create -f - &>/dev/null <<EOF || rv=$?
apiVersion: build.openshift.io/v1
kind: BuildConfig
metadata:
  name: $IMAGENAME-$TAG
spec:
  output:
    to:
      kind: ImageStreamTag
      name: $IMAGENAME:$TAG
  source:
    type: Binary
  strategy:
    dockerStrategy:
      dockerfilePath: $DOCKERFILE
      forcePull: true
    type: Docker
EOF

  if [[ $rv -eq 0 ]]; then
    tar -cz . | oc start-build $IMAGENAME-$TAG --from-archive=- -F
  else
    for ((i=0; i<60; i++)); do
      oc get istag $IMAGENAME:$TAG &>/dev/null && break
      sleep 10
    done
    oc get istag $IMAGENAME:$TAG &>/dev/null
  fi
fi
