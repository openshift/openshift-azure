#!/bin/bash

set -eo pipefail

if [[ -n "$ARTIFACTS" ]]; then
  ARTIFACT_FLAG="-artifact-dir=$ARTIFACTS"
fi

if [[ -n "$FOCUS" ]]; then
    FOCUS="-ginkgo.focus=$FOCUS"
fi

if [[ -z "$TIMEOUT" ]]; then
    TIMEOUT=20m
fi

# start the fake rp server if needed
if [[ "$FOCUS" == *"\[Fake\]"* ]]; then
    go generate ./...
    go run cmd/fakerp/main.go &
    while [[ "$(curl -s -o /dev/null -w '%{http_code}' localhost:8080)" == "000" ]]; do sleep 2; done
    trap 'return_id=$?; set +ex; kill $(lsof -t -i :8080); wait $(lsof -t -i :8080); exit $return_id' EXIT
fi

go test \
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
