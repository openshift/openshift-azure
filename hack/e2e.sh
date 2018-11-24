#!/bin/bash

set -eo pipefail

if [[ -n "$ARTIFACT_DIR" ]]; then
  ARTIFACT_FLAG="-artifact-dir=$ARTIFACT_DIR"
fi

if [[ -n "$FOCUS" ]]; then
    FOCUS="-ginkgo.focus=$FOCUS"
fi

go test ./test/e2e -timeout 20m -test.v -ginkgo.v "${FOCUS:-}" -ginkgo.noColor -tags e2e "${ARTIFACT_FLAG:-}"

# -ldflags "-X github.com/openshift/openshift-azure/test/e2e.gitCommit=COMMIT"
