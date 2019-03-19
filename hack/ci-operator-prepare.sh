#!/bin/bash -ex

export ARTIFACT_DIR=/tmp/artifacts
export GOPATH=/go # our prow configuration overrides our image setting to /home/prow/go

ln -s /usr/local/e2e-secrets/azure secrets
set +x
. ./secrets/secret
az login --service-principal -u ${AZURE_CLIENT_ID} -p ${AZURE_CLIENT_SECRET} --tenant ${AZURE_TENANT_ID} >/dev/null
set -x

export SYNC_IMAGE=registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:sync
export ETCDBACKUP_IMAGE=registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:etcdbackup
export AZURE_CONTROLLERS_IMAGE=registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:azure-controllers
export METRICSBRIDGE_IMAGE=registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:metricsbridge
export STARTUP_IMAGE=registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:startup
export TLSPROXY_IMAGE=registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tlsproxy

export NO_WAIT=true
