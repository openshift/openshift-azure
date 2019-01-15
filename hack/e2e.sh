#!/bin/bash

set -eo pipefail

cleanup() {
    kill $(jobs -p) &>/dev/null || true
    wait
}

trap cleanup EXIT

if [[ -n "$ARTIFACT_DIR" ]]; then
  ARTIFACT_FLAG="-artifact-dir=$ARTIFACT_DIR"
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
