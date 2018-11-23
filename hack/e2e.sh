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
  # Login as osadmin to simulate a regular user
  password=$(awk '/^  endUserPasswd:/ { print $2 }' <_data/containerservice.yaml)
  fqdn=$(awk '/^  fqdn:/ { print $2 }' <_data/containerservice.yaml)
  export KUBECONFIG=$(pwd)/_data/_out/enduser.kubeconfig
  oc login $fqdn --username enduser --password $password --insecure-skip-tls-verify=true
  go test ./test/suites/enduser -tags e2e -test.v -ginkgo.v -timeout 20m -ginkgo.focus="\[EndUser\]" -ginkgo.noColor -ginkgo.randomizeAllSpecs "${ARTIFACT_FLAG:-}"
  oc logout
fi

if [[ -z "$SUITE" || "$SUITE" == "clusterreader" ]]; then
  echo "Running azure cluster reader e2e tests"
  (awk '/^  azureClusterReaderKubeconfig:/ { print $2 }' <_data/containerservice.yaml | base64 -d) > $(pwd)/_data/_out/azure-cluster-reader.kubeconfig
  export KUBECONFIG=$(pwd)/_data/_out/azure-cluster-reader.kubeconfig
  go test ./test/suites/azurereader -tags e2e -test.v -ginkgo.v -timeout 20m -ginkgo.focus="\[AzureClusterReader\]" -ginkgo.noColor -ginkgo.randomizeAllSpecs "${ARTIFACT_FLAG:-}"
fi

if [[ -z "$SUITE" || "$SUITE" == "customer-cluster-admin" ]]; then
  fqdn=$(awk '/^  fqdn:/ { print $2 }' <_data/containerservice.yaml)

  echo "Running azure customer-cluster-admin tests"
  export KUBECONFIG=$(pwd)/_data/_out/customer-cluster-admin.kubeconfig
  oc login $fqdn --username customer-cluster-admin --password "$(awk '/^  customerAdminPasswd:/{ print $2 }' <_data/containerservice.yaml)" --insecure-skip-tls-verify=true
  go test ./test/suites/customeradmin -tags e2e -test.v -ginkgo.v -timeout 20m -ginkgo.noColor "${ARTIFACT_FLAG:-}"

  export KUBECONFIG=$(pwd)/_data/_out/customer-cluster-reader.kubeconfig
  oc login $fqdn --username customer-cluster-reader --password "$(awk '/^  customerReaderPasswd:/{ print $2 }' <_data/containerservice.yaml)" --insecure-skip-tls-verify=true
  go test ./test/suites/customerreader -tags e2e -test.v -ginkgo.v -timeout 20m -ginkgo.noColor "${ARTIFACT_FLAG:-}"

  export KUBECONFIG=$(pwd)/_data/_out/enduser.kubeconfig
  oc login $fqdn --username enduser --password "$(awk '/^  endUserPasswd:/{ print $2 }' <_data/containerservice.yaml)" --insecure-skip-tls-verify=true
  go test ./test/suites/enduser -tags e2e -test.v -ginkgo.v -timeout 20m -ginkgo.noColor -ginkgo.randomizeAllSpecs "${ARTIFACT_FLAG:-}"
fi

if [[ "$SUITE" == "keyrotation" ]]; then
  echo "Running key rotation e2e tests"
  export KUBECONFIG=$(pwd)/_data/_out/admin.kubeconfig
  go test ./test/suites/keyrotation -tags e2e -test.v -ginkgo.v -timeout 80m -ginkgo.focus="Fake" -ginkgo.noColor "${ARTIFACT_FLAG:-}"
fi

if [[ "$SUITE" == "scaleupdown" ]]; then
  echo "Running scale up/down e2e tests"
  export KUBECONFIG=$(pwd)/_data/_out/admin.kubeconfig
  go test ./test/suites/scaleupdown -tags e2e -test.v -ginkgo.v -timeout 20m -ginkgo.focus="Fake" -ginkgo.noColor "${ARTIFACT_FLAG:-}"
fi
