#!/bin/bash -e

set -o pipefail

. hack/tests/ci-prepare.sh

make unit

CI_SERVER_URL=https://openshift-gce-devel.appspot.com/build/origin-ci-test

# Configure the git refs and job link based on how the job was triggered via prow
if [[ "$JOB_TYPE" == "presubmit" ]]; then
	echo "detected PR code coverage job for #$PULL_NUMBER"
	REF_FLAGS="-P $PULL_NUMBER -C $PULL_BASE_SHA"
	JOB_LINK=$CI_SERVER_URL/pr-logs/pull/${REPO_OWNER}_$REPO_NAME/$PULL_NUMBER/$JOB_NAME/$BUILD_ID
elif [[ "$JOB_TYPE" == "postsubmit" ]]; then
	echo "detected branch code coverage job for ${PULL_BASE_REF}"
	REF_FLAGS="-B $PULL_BASE_REF -C $PULL_BASE_SHA"
	JOB_LINK=$CI_SERVER_URL/logs/$JOB_NAME/$BUILD_ID
else
	echo "$JOB_TYPE jobs not supported"
	exit 1
fi

# Configure certain internal codecov variables with values from prow.
export CI_BUILD_URL=$JOB_LINK
export CI_BUILD_ID=$JOB_NAME
export CI_JOB_ID=$BUILD_ID

# bash <(curl -s https://codecov.io/bash) -Z -f coverage.out -r $REPO_OWNER/$REPO_NAME $REF_FLAGS
env
