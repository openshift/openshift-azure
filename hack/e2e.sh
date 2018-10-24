#!/bin/bash

set -eo pipefail

echo "SUITE=>${SUITE}"
if [[ -z "$RESOURCEGROUP" ]]; then
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

if [[ -n "$ARTIFACT_DIR" ]]; then
  ARTIFACT_FLAG="-artifact-dir=$ARTIFACT_DIR"
fi

if [[ -z "$SUITE" || "$SUITE" == "enduser" ]]; then
  echo "Running end user e2e tests"
  # Login as enduser to simulate a regular user
  password=$(awk '/^  endUserPasswd:/ { print $2 }' <_data/containerservice.yaml)
  fqdn=$(awk '/^  fqdn:/ { print $2 }' <_data/containerservice.yaml)
  export KUBECONFIG=$(pwd)/_data/_out/enduser.kubeconfig
  # oc login is going to create the enduser.kubeconfig for us
  oc login $fqdn --username enduser --password $password --insecure-skip-tls-verify=true
  go test ./test/e2e -test.v -ginkgo.v -ginkgo.focus="\[EndUser\]" -ginkgo.noColor -ginkgo.randomizeAllSpecs -tags e2e "${ARTIFACT_FLAG:-}"
  oc logout
fi

if [[ -z "$SUITE" || "$SUITE" == "clusterreader" ]]; then
  echo "Running azure cluster reader e2e tests"
  (awk '/^  azureClusterReaderKubeconfig:/ { print $2 }' <_data/containerservice.yaml | base64 -d) > _data/_out/azure-cluster-reader.kubeconfig

  export KUBECONFIG=$(pwd)/_data/_out/azure-cluster-reader.kubeconfig
  go test ./test/e2e -test.v -ginkgo.v -ginkgo.focus="\[AzureClusterReader\]" -ginkgo.noColor -ginkgo.randomizeAllSpecs -tags e2e
fi

if [[ -z "$SUITE" || "$SUITE" == "customer-cluster-admin" ]]; then
  echo "Running azure customer-cluster-admin tests"
  fqdn=$(awk '/^  fqdn:/ { print $2 }' <_data/containerservice.yaml)

  export KUBECONFIG=$(pwd)/_data/_out/customer-cluster-admin.kubeconfig
  oc login $fqdn --username customer-cluster-admin --password "$(awk '/^  customerAdminPasswd:/{ print $2 }' <_data/containerservice.yaml)" --insecure-skip-tls-verify=true
  oc config rename-context $(oc config current-context) customer-cluster-admin

  export KUBECONFIG=$(pwd)/_data/_out/customer-cluster-reader.kubeconfig
  oc login $fqdn --username customer-cluster-reader --password "$(awk '/^  customerReaderPasswd:/{ print $2 }' <_data/containerservice.yaml)" --insecure-skip-tls-verify=true
  oc config rename-context $(oc config current-context) customer-cluster-reader

  export KUBECONFIG=$(pwd)/_data/_out/customer-cluster-enduser.kubeconfig
  oc login $fqdn --username enduser --password "$(awk '/^  endUserPasswd:/{ print $2 }' <_data/containerservice.yaml)" --insecure-skip-tls-verify=true
  oc config rename-context $(oc config current-context) enduser

  export KUBECONFIG=$(pwd)/_data/_out/customer-cluster-admin.kubeconfig:$(pwd)/_data/_out/customer-cluster-reader.kubeconfig:$(pwd)/_data/_out/customer-cluster-enduser.kubeconfig
  go test ./test/e2e -test.v -ginkgo.v -ginkgo.focus="\[CustomerAdmin\]" -ginkgo.noColor -ginkgo.randomizeAllSpecs -tags e2e "${ARTIFACT_FLAG:-}"
fi
