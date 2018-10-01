#!/bin/bash -e

if [[ -z "$RESOURCEGROUP" ]]; then
    RESOURCEGROUP=$(awk '/^    resourceGroup:/ { print $2 }' <_data/containerservice.yaml)
fi

echo "Running end user e2e tests"
# Login as osadmin to simulate a regural user
password=$(hack/config.sh get-config $RESOURCEGROUP | jq -r .config.adminPasswd)
fqdn=$(hack/config.sh get-config $RESOURCEGROUP | jq -r .properties.fqdn)
oc login $fqdn --username osadmin --password $password --insecure-skip-tls-verify=true
oc new-project e2e-end-user-test-root
# TODO: Run the e2e image inside a job. Figure out whether we run as part of ci-operator
# or it's just a local run.

# TODO: Wait for the job to finish, report results
oc delete project e2e-end-user-test-root
oc logout

echo "Running azure cluster reader e2e tests"
(hack/config.sh get-config $RESOURCEGROUP | jq -r .config.azureClusterReaderKubeconfig | base64 -d) > _data/_out/azure-cluster-reader.kubeconfig
go test ./test/e2e -ginkgo.focus="\[AzureClusterReader\]" -tags e2e -kubeconfig ../../_data/_out/azure-cluster-reader.kubeconfig
