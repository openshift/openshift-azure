#!/bin/bash -e

if [[ -z "$RESOURCEGROUP" ]]; then
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

if [ -z "$SUITE" ] || [ "$SUITE" == "enduser" ]; then
  echo "Running end user e2e tests"
  # Login as osadmin to simulate a regular user
  password=$(awk '/^  adminPasswd:/ { print $2 }' <_data/containerservice.yaml)
  fqdn=$(awk '/^  fqdn:/ { print $2 }' <_data/containerservice.yaml)
  export KUBECONFIG=_data/_out/osadmin.kubeconfig
  # oc login is going to create the osadmin.kubeconfig for us
  oc login $fqdn --username osadmin --password $password --insecure-skip-tls-verify=true
  go test ./test/e2e -test.v -ginkgo.v -ginkgo.focus="\[EndUser\]" -ginkgo.noColor -ginkgo.randomizeAllSpecs -tags e2e -kubeconfig ../../_data/_out/osadmin.kubeconfig 2>&1 | tee end-user.log
  oc logout
fi;

if [ -z "$SUITE" ] || [ "$SUITE" == "clusterreader" ]; then
  echo "Running azure cluster reader e2e tests"
  (awk '/^  azureClusterReaderKubeconfig:/ { print $2 }' <_data/containerservice.yaml | base64 -d) > _data/_out/azure-cluster-reader.kubeconfig
  go test ./test/e2e -test.v -ginkgo.v -ginkgo.focus="\[AzureClusterReader\]" -ginkgo.noColor -ginkgo.randomizeAllSpecs -tags e2e -kubeconfig ../../_data/_out/azure-cluster-reader.kubeconfig 2>&1 | tee azure-reader.log
fi
