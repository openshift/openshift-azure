#!/bin/bash

set -eo pipefail

if [[ -z "${CODECOV_UPLOAD_TOKEN}" ]]; then
    echo "CODECOV_UPLOAD_TOKEN must be set"
    exit 1
fi

bash <(curl -s https://codecov.io/bash) -Z -K -t ${CODECOV_UPLOAD_TOKEN} -f coverage.out -r ${REPO_OWNER}/${REPO_NAME} -P ${PULL_NUMBER} -C ${PULL_BASE_SHA}
