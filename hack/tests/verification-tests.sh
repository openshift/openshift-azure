#!/bin/bash -e
# Get verification tests source code prepare it and run the verification tests (Bushslicer)

set -o pipefail

. hack/tests/ci-prepare.sh

function get_az_hosts() {
  local IFS=$'\n'
  for ip in $(az vmss list-instance-public-ips -g $RESOURCEGROUP -n ss-master --query "[].ipAddress" -o tsv) ; do
    i+=${ip}':master:node,'
  done
  echo ${i%?}
}

BRANCH="${BRANCH:-v3}"
VERIFICATION_TESTS_GIT="${VERIFICATION_TESTS_GIT:-https://github.com/openshift/verification-tests.git}"
# Env vars for setting up test
ARO_PUBLIC_HOSTNAME="${ARO_PUBLIC_HOSTNAME:-$(awk '/^\s+publicHostname:/{ print $2}' ${HOME}/go/src/github.com/openshift/openshift-azure/_data/containerservice.yaml)}"
RESOURCEGROUP="${RESOURCEGROUP:-$(awk '/^name:/{ print $2}' ${HOME}/go/src/github.com/openshift/openshift-azure/_data/containerservice.yaml)}"
export OPENSHIFT_ENV_OSE_WEB_CONSOLE_URL="${OPENSHIFT_ENV_OSE_WEB_CONSOLE_URL:-https://${ARO_PUBLIC_HOSTNAME}/}"
export BUSHSLICER_CONFIG='{"global": {"browser": "chrome"}}'
export BUSHSLICER_DEFAULT_ENVIRONMENT="${BUSHSLICER_DEFAULT_ENVIRONMENT:-ose}"
export OPENSHIFT_ENV_OSE_API_PORT="${OPENSHIFT_ENV_OSE_API_PORT:-443}"
export OPENSHIFT_ENV_SERVICES_AZURE_HOST_CONNECT_OPS_USER="${OPENSHIFT_ENV_SERVICES_AZURE_HOST_CONNECT_OPS_USER:-cloud-user}"
export OPENSHIFT_ENV_SERVICES_AZURE_HOST_CONNECT_OPS_SSH_PRIVATE_KEY="${OPENSHIFT_ENV_SERVICES_AZURE_HOST_CONNECT_OPS_SSH_PRIVATE_KEY:-${HOME}/go/src/github.com/openshift/openshift-azure/_data/_out/id_rsa}"
export OPENSHIFT_ENV_OSE_USER_MANAGER_USERS="${OPENSHIFT_ENV_OSE_USER_MANAGER_USERS:-:MYLOGINTOKEN}"
export OPENSHIFT_ENV_OSE_HOSTS="${OPENSHIFT_ENV_OSE_HOSTS:-$(get_az_hosts)}"
# must be URL to new line separated case IDs in format OCP-xxxx
TESTS_URL="${TESTS_URL:-http://aosqe-test-lists.s3-website-us-east-1.amazonaws.com/lists/3.11_ded_azure.lst}"
echo =========================
env
echo =========================

git clone --depth 1 --branch $BRANCH $VERIFICATION_TESTS_GIT
pushd verification-tests
tools/openshift-ci/verification_tests_ci_entrypoint.sh $TESTS_URL
# do what we need to do to publish logs under junit-report/
