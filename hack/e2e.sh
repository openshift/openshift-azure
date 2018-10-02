#!/bin/bash -e

if [[ -z "$RESOURCEGROUP" ]]; then
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

echo "Running end user e2e tests"
# Login as osadmin to simulate a regural user
password=$(hack/config.sh get-config $RESOURCEGROUP | jq -r .config.adminPasswd)
fqdn=$(hack/config.sh get-config $RESOURCEGROUP | jq -r .properties.fqdn)
export KUBECONFIG=_data/_out/osadmin.kubeconfig
# oc login is going to create the osadmin.kubeconfig for us
oc login $fqdn --username osadmin --password $password --insecure-skip-tls-verify=true
go test ./test/e2e -test.v -ginkgo.v -ginkgo.focus="\[EndUser\]" -tags e2e -kubeconfig ../../_data/_out/osadmin.kubeconfig
oc logout

echo "Running azure cluster reader e2e tests"
(hack/config.sh get-config $RESOURCEGROUP | jq -r .config.azureClusterReaderKubeconfig | base64 -d) > _data/_out/azure-cluster-reader.kubeconfig
go test ./test/e2e -test.v -ginkgo.v -ginkgo.focus="\[AzureClusterReader\]" -tags e2e -kubeconfig ../../_data/_out/azure-cluster-reader.kubeconfig
