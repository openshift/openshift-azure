#!/bin/bash -e

set -o pipefail

if [[ -n "$ARTIFACTS" ]]; then
  ARTIFACT_FLAG="-artifact-dir=$ARTIFACTS"
fi

if [[ -n "$FOCUS" ]]; then
    FOCUS="-ginkgo.focus=$FOCUS"
fi

if [[ -z "$TIMEOUT" ]]; then
    TIMEOUT=60m
fi

[[ -e /var/run/secrets/kubernetes.io ]] || go generate ./...
# building takes time, so if we use "go run" we will not get the PID of
# the server as it won't have started yet.
go build ./cmd/fakerp
./fakerp &

trap 'return_id=$?; set +ex; kill $(lsof -t -i :8080); wait $(lsof -t -i :8080); exit $return_id' EXIT
while [[ "$(curl -s -o /dev/null -w '%{http_code}' localhost:8080)" == "000" ]]; do sleep 2; done

go test \
-race \
-ldflags "-X github.com/openshift/openshift-azure/test/e2e.gitCommit=COMMIT" \
./test/e2e \
-timeout "$TIMEOUT" \
-v \
-ginkgo.v \
"${FOCUS:-}" \
-ginkgo.noColor \
-tags e2e \
"${ARTIFACT_FLAG:-}" \
"${EXTRA_FLAGS:-}"
